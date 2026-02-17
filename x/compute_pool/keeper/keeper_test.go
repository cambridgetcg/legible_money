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

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/legible-money/legible/x/compute_pool/keeper"
	"github.com/legible-money/legible/x/compute_pool/types"
)

// moduleAddrStr returns the bech32 module address for balance lookups.
func moduleAddrStr() string {
	return authtypes.NewModuleAddress(types.ModuleName).String()
}

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) setBalance(addr string, denom string, amount sdkmath.Int) {
	m.balances[addr+"/"+denom] = amount
}

func (m *mockBankKeeper) getBal(key string) sdkmath.Int {
	bal, ok := m.balances[key]
	if !ok {
		return sdkmath.ZeroInt()
	}
	return bal
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := fromAddr.String() + "/" + coin.Denom
		toKey := toAddr.String() + "/" + coin.Denom
		fromBal := m.getBal(fromKey)
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)
		m.balances[toKey] = m.getBal(toKey).Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) modAddrKey(moduleName string, denom string) string {
	return authtypes.NewModuleAddress(moduleName).String() + "/" + denom
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal := m.getBal(key)
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		modKey := m.modAddrKey(recipientModule, coin.Denom)
		m.balances[modKey] = m.getBal(modKey).Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		modKey := m.modAddrKey(senderModule, coin.Denom)
		modBal := m.getBal(modKey)
		if modBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[modKey] = modBal.Sub(coin.Amount)
		key := recipientAddr.String() + "/" + coin.Denom
		m.balances[key] = m.getBal(key).Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := m.modAddrKey(senderModule, coin.Denom)
		toKey := m.modAddrKey(recipientModule, coin.Denom)
		fromBal := m.getBal(fromKey)
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)
		m.balances[toKey] = m.getBal(toKey).Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	key := addr.String() + "/" + denom
	bal, ok := m.balances[key]
	if !ok {
		return sdk.NewCoin(denom, sdkmath.ZeroInt())
	}
	return sdk.NewCoin(denom, bal)
}

func (m *mockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	key := addr.String() + "/ulgm"
	bal, ok := m.balances[key]
	if !ok {
		return sdk.Coins{}
	}
	return sdk.NewCoins(sdk.NewCoin("ulgm", bal))
}

// -----------------------------------------------------------------------
// Mock StakingKeeper
// -----------------------------------------------------------------------

type mockStakingKeeper struct {
	validators map[string]struct {
		tier  string
		stake string
	}
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		validators: make(map[string]struct {
			tier  string
			stake string
		}),
	}
}

func (m *mockStakingKeeper) setValidator(addr, tier, stake string) {
	m.validators[addr] = struct {
		tier  string
		stake string
	}{tier: tier, stake: stake}
}

func (m *mockStakingKeeper) GetValidatorByAddress(ctx sdk.Context, addr string) (tier string, stake string, found bool) {
	v, ok := m.validators[addr]
	if !ok {
		return "", "", false
	}
	return v.tier, v.stake, true
}

// -----------------------------------------------------------------------
// Mock KnowledgeKeeper
// -----------------------------------------------------------------------

type mockKnowledgeKeeper struct{}

func (m *mockKnowledgeKeeper) GetClaimComplexity(ctx sdk.Context, claimId string) (uint64, uint64, bool) {
	return 2, 1, true // 2 evidence, 1 cross-domain ref
}

// -----------------------------------------------------------------------
// Test setup
// -----------------------------------------------------------------------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockStakingKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	storeService := runtime.NewKVStoreService(storeKey)

	bankKeeper := newMockBankKeeper()
	stakingKeeper := newMockStakingKeeper()
	knowledgeKeeper := &mockKnowledgeKeeper{}

	k := keeper.NewKeeper(
		cdc,
		storeService,
		bankKeeper,
		stakingKeeper,
		knowledgeKeeper,
		"legible1govauthority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Initialize with default params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, bankKeeper, stakingKeeper
}

// Test addresses
var (
	providerAddr1 = sdk.AccAddress([]byte("provider1___________"))
	providerAddr2 = sdk.AccAddress([]byte("provider2___________"))
	validatorAddr = sdk.AccAddress([]byte("validator1__________"))
)

// -----------------------------------------------------------------------
// Tests: Params
// -----------------------------------------------------------------------

func TestDefaultParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.ComputePoolShareBps != types.DefaultComputePoolShareBps {
		t.Errorf("expected ComputePoolShareBps=%d, got %d", types.DefaultComputePoolShareBps, params.ComputePoolShareBps)
	}
	if params.BaseCUPerVerification != 100 {
		t.Errorf("expected BaseCUPerVerification=100, got %d", params.BaseCUPerVerification)
	}
	if params.MinProviderStake != "100000000" {
		t.Errorf("expected MinProviderStake=100000000, got %s", params.MinProviderStake)
	}
}

func TestParamsValidation(t *testing.T) {
	params := types.DefaultParams()
	if err := params.Validate(); err != nil {
		t.Errorf("default params should validate: %v", err)
	}

	// Invalid: pool share too high
	bad := params
	bad.ComputePoolShareBps = 1000001
	if err := bad.Validate(); err == nil {
		t.Error("expected error for pool share > 1000000")
	}

	// Invalid: zero base CU
	bad = params
	bad.BaseCUPerVerification = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero base CU")
	}

	// Invalid: bad min stake
	bad = params
	bad.MinProviderStake = "abc"
	if err := bad.Validate(); err == nil {
		t.Error("expected error for invalid min stake")
	}
}

// -----------------------------------------------------------------------
// Tests: Provider Registration
// -----------------------------------------------------------------------

func TestRegisterProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	// Fund the provider account
	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
		Did:         "did:lgm:provider1",
	}

	err := k.RegisterProvider(ctx, msg)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	// Verify provider was stored
	provider, found := k.GetProvider(ctx, providerAddr1.String())
	if !found {
		t.Fatal("provider not found after registration")
	}
	if provider.Status != "active" {
		t.Errorf("expected status=active, got %s", provider.Status)
	}
	if provider.PricePerCU != "5000" {
		t.Errorf("expected price=5000, got %s", provider.PricePerCU)
	}
	if provider.Stake != "100000000" {
		t.Errorf("expected stake=100000000, got %s", provider.Stake)
	}

	// Verify stake was deducted
	remaining := bk.balances[providerAddr1.String()+"/ulgm"]
	if !remaining.Equal(sdkmath.NewInt(100000000)) {
		t.Errorf("expected remaining balance=100000000, got %s", remaining.String())
	}
}

func TestRegisterProviderInsufficientStake(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(50000000)) // less than 100 LGM

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "50000000",
	}

	err := k.RegisterProvider(ctx, msg)
	if err == nil {
		t.Fatal("expected error for insufficient stake")
	}
}

func TestRegisterProviderDuplicate(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(300000000))

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}

	_ = k.RegisterProvider(ctx, msg)
	err := k.RegisterProvider(ctx, msg)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegisterProviderPriceExceedsCeiling(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "99999999", // way above ceiling of 10000
		StakeAmount: "100000000",
	}

	err := k.RegisterProvider(ctx, msg)
	if err == nil {
		t.Fatal("expected error for price exceeding ceiling")
	}
}

// -----------------------------------------------------------------------
// Tests: Provider Lifecycle
// -----------------------------------------------------------------------

func TestProviderDeregistration(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	// Register
	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, msg)

	// Deregister
	err := k.DeregisterProviderState(ctx, providerAddr1.String())
	if err != nil {
		t.Fatalf("failed to deregister: %v", err)
	}

	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.Status != "unbonding" {
		t.Errorf("expected status=unbonding, got %s", provider.Status)
	}
}

func TestProviderHeartbeat(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, msg)

	// Send heartbeat at a later block
	ctx = ctx.WithBlockHeight(200)
	err := k.ProcessHeartbeat(ctx, providerAddr1.String())
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.LastHeartbeat != 200 {
		t.Errorf("expected last_heartbeat=200, got %d", provider.LastHeartbeat)
	}
}

func TestProviderLivenessCheck(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, msg)

	// Advance past heartbeat interval (1000 blocks)
	ctx = ctx.WithBlockHeight(1200)
	k.CheckProviderLiveness(ctx)

	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.Status != "suspended" {
		t.Errorf("expected status=suspended after missed heartbeat, got %s", provider.Status)
	}

	// Heartbeat should re-activate
	_ = k.ProcessHeartbeat(ctx, providerAddr1.String())
	provider, _ = k.GetProvider(ctx, providerAddr1.String())
	if provider.Status != "active" {
		t.Errorf("expected status=active after heartbeat, got %s", provider.Status)
	}
}

func TestProviderPauseUnpause(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	// Pause
	updateMsg := &types.MsgUpdateProvider{
		Sender: providerAddr1.String(),
		Pause:  true,
	}
	_ = k.UpdateProviderState(ctx, updateMsg)

	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.Status != "paused" {
		t.Errorf("expected status=paused, got %s", provider.Status)
	}

	// Unpause
	updateMsg = &types.MsgUpdateProvider{
		Sender:  providerAddr1.String(),
		Unpause: true,
	}
	_ = k.UpdateProviderState(ctx, updateMsg)

	provider, _ = k.GetProvider(ctx, providerAddr1.String())
	if provider.Status != "active" {
		t.Errorf("expected status=active after unpause, got %s", provider.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: Credit Calculation
// -----------------------------------------------------------------------

func TestCalculateCredits(t *testing.T) {
	params := types.DefaultParams()

	tests := []struct {
		name            string
		tier            string
		evidence        uint64
		crossDomain     uint64
		expectedCredits uint64
	}{
		{
			name:            "simple apprentice",
			tier:            "apprentice",
			evidence:        0,
			crossDomain:     0,
			expectedCredits: 50, // 100 × 1.0 × 0.5
		},
		{
			name:            "simple bonded",
			tier:            "bonded",
			evidence:        0,
			crossDomain:     0,
			expectedCredits: 100, // 100 × 1.0 × 1.0
		},
		{
			name:            "complex guardian",
			tier:            "guardian",
			evidence:        3,
			crossDomain:     2,
			expectedCredits: 175, // 100 × 1175000/1000000 × 1500/1000 = 117 × 1.5 = 175
		},
		{
			name:            "verified with evidence",
			tier:            "verified",
			evidence:        2,
			crossDomain:     0,
			expectedCredits: 84, // 100 × 1050000/1000000 × 800/1000 = 105 × 0.8 = 84
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := keeper.CalculateCredits(params, tc.tier, tc.evidence, tc.crossDomain)
			if got != tc.expectedCredits {
				t.Errorf("expected %d, got %d", tc.expectedCredits, got)
			}
		})
	}
}

func TestTierMultiplier(t *testing.T) {
	tests := []struct {
		tier     string
		expected uint64
	}{
		{"apprentice", 500},
		{"verified", 800},
		{"bonded", 1000},
		{"guardian", 1500},
		{"unknown", 500},
	}

	for _, tc := range tests {
		got := keeper.TierMultiplierBps(tc.tier)
		if got != tc.expected {
			t.Errorf("tier %s: expected %d, got %d", tc.tier, tc.expected, got)
		}
	}
}

// -----------------------------------------------------------------------
// Tests: Credit Granting
// -----------------------------------------------------------------------

func TestGrantCredits(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Grant credits for a verification
	cuAmount, err := k.GrantCredits(ctx, validatorAddr.String(), "claim-123", 1, 0)
	if err != nil {
		t.Fatalf("failed to grant credits: %v", err)
	}

	// bonded tier with 1 evidence, 0 cross-domain:
	// complexityBps = 1000000 + 25000*1 = 1025000
	// credits = 100 * 1025000 / 1000000 = 102, × 1000/1000 = 102
	if cuAmount != 102 {
		t.Errorf("expected 102 CU, got %d", cuAmount)
	}

	credit, found := k.GetCredit(ctx, validatorAddr.String())
	if !found {
		t.Fatal("credit not found")
	}
	if credit.Balance != 102 {
		t.Errorf("expected balance=102, got %d", credit.Balance)
	}
	if credit.EarnedTotal != 102 {
		t.Errorf("expected earned_total=102, got %d", credit.EarnedTotal)
	}
}

func TestGrantCreditsAccumulate(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Grant twice
	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0)
	k.GrantCredits(ctx, validatorAddr.String(), "claim-2", 0, 0)

	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	// 100 + 100 = 200
	if credit.Balance != 200 {
		t.Errorf("expected balance=200, got %d", credit.Balance)
	}
}

func TestGrantCreditsMaxBalance(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Max balance = 100 × 10 × 1000/1000 = 1000
	// Grant 11 times = 1100 CU, should cap at 1000
	for i := 0; i < 11; i++ {
		k.GrantCredits(ctx, validatorAddr.String(), fmt.Sprintf("claim-%d", i), 0, 0)
	}

	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 1000 {
		t.Errorf("expected balance capped at 1000, got %d", credit.Balance)
	}
}

// -----------------------------------------------------------------------
// Tests: Credit Expiration
// -----------------------------------------------------------------------

func TestCreditExpiration(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Grant credits at block 100
	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0)

	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 100 {
		t.Fatalf("expected balance=100, got %d", credit.Balance)
	}

	// Advance past expiry (100 + 34272 + 1 = 34373)
	ctx = ctx.WithBlockHeight(34373)
	k.ExpireCredits(ctx)

	credit, _ = k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 0 {
		t.Errorf("expected balance=0 after expiry, got %d", credit.Balance)
	}
	if credit.ExpiredTotal != 100 {
		t.Errorf("expected expired_total=100, got %d", credit.ExpiredTotal)
	}
}

// -----------------------------------------------------------------------
// Tests: Full Redemption Lifecycle
// -----------------------------------------------------------------------

func TestRedemptionLifecycle(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	// Setup: register provider, grant credits, fund pool
	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(500000000)) // 500 LGM in pool

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Register provider
	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.provider1.example.com",
		PricePerCU:  "1000", // 0.001 LGM per CU
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	// Grant credits
	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // 100 CU

	// Create redemption ticket
	ticket, err := k.CreateRedemptionTicket(ctx, validatorAddr.String(), providerAddr1.String(), 50)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if ticket.CUAmount != 50 {
		t.Errorf("expected cu_amount=50, got %d", ticket.CUAmount)
	}
	if ticket.LGMAmount != "50000" { // 50 × 1000
		t.Errorf("expected lgm_amount=50000, got %s", ticket.LGMAmount)
	}
	if ticket.Status != "pending" {
		t.Errorf("expected status=pending, got %s", ticket.Status)
	}

	// Credits should be deducted
	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 50 {
		t.Errorf("expected remaining credits=50, got %d", credit.Balance)
	}

	// Confirm usage
	err = k.ConfirmRedemption(ctx, validatorAddr.String(), ticket.Id, 50)
	if err != nil {
		t.Fatalf("failed to confirm: %v", err)
	}

	// Check ticket is confirmed
	confirmedTicket, _ := k.GetTicket(ctx, ticket.Id)
	if confirmedTicket.Status != "confirmed" {
		t.Errorf("expected status=confirmed, got %s", confirmedTicket.Status)
	}

	// Check usage report was created
	report, found := k.GetUsageReport(ctx, ticket.Id)
	if !found {
		t.Fatal("usage report not found")
	}
	if report.ActualCU != 50 {
		t.Errorf("expected actual_cu=50, got %d", report.ActualCU)
	}
	if report.LGMPaid != "50000" {
		t.Errorf("expected lgm_paid=50000, got %s", report.LGMPaid)
	}
}

func TestRedemptionPartialUsage(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(types.ModuleName, "ulgm", sdkmath.NewInt(500000000))

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "1000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // 100 CU

	ticket, _ := k.CreateRedemptionTicket(ctx, validatorAddr.String(), providerAddr1.String(), 100)

	// Confirm with only 30 actual CU
	err := k.ConfirmRedemption(ctx, validatorAddr.String(), ticket.Id, 30)
	if err != nil {
		t.Fatalf("failed to confirm partial: %v", err)
	}

	// 70 CU should be returned
	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 70 {
		t.Errorf("expected 70 CU returned, got balance=%d", credit.Balance)
	}

	// Report should show actual amount
	report, _ := k.GetUsageReport(ctx, ticket.Id)
	if report.ActualCU != 30 {
		t.Errorf("expected actual_cu=30, got %d", report.ActualCU)
	}
	if report.LGMPaid != "30000" {
		t.Errorf("expected lgm_paid=30000, got %s", report.LGMPaid)
	}
}

func TestRedemptionDispute(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(types.ModuleName, "ulgm", sdkmath.NewInt(500000000))

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "1000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // 100 CU

	ticket, _ := k.CreateRedemptionTicket(ctx, validatorAddr.String(), providerAddr1.String(), 50)

	// Dispute
	err := k.DisputeRedemption(ctx, validatorAddr.String(), ticket.Id, "service not delivered")
	if err != nil {
		t.Fatalf("failed to dispute: %v", err)
	}

	// Credits should be returned
	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 100 { // 50 remaining + 50 returned
		t.Errorf("expected 100 CU after dispute, got %d", credit.Balance)
	}

	// Provider should have tasks_failed incremented
	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.TasksFailed != 1 {
		t.Errorf("expected tasks_failed=1, got %d", provider.TasksFailed)
	}

	// Ticket should be disputed
	disputedTicket, _ := k.GetTicket(ctx, ticket.Id)
	if disputedTicket.Status != "disputed" {
		t.Errorf("expected status=disputed, got %s", disputedTicket.Status)
	}
}

func TestRedemptionExpiry(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(types.ModuleName, "ulgm", sdkmath.NewInt(500000000))

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "1000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // 100 CU

	ticket, _ := k.CreateRedemptionTicket(ctx, validatorAddr.String(), providerAddr1.String(), 50)

	// Advance past ticket expiry (100 + 1000 + 1)
	ctx = ctx.WithBlockHeight(1101)
	k.ExpireRedemptionTickets(ctx)

	// Credits should be returned
	credit, _ := k.GetCredit(ctx, validatorAddr.String())
	if credit.Balance != 100 {
		t.Errorf("expected 100 CU after expiry, got %d", credit.Balance)
	}

	// Ticket should be expired
	expiredTicket, _ := k.GetTicket(ctx, ticket.Id)
	if expiredTicket.Status != "expired" {
		t.Errorf("expected status=expired, got %s", expiredTicket.Status)
	}
}

func TestRedemptionInsufficientCredits(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(types.ModuleName, "ulgm", sdkmath.NewInt(500000000))

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "1000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // 100 CU

	// Try to redeem more than available
	_, err := k.CreateRedemptionTicket(ctx, validatorAddr.String(), providerAddr1.String(), 200)
	if err == nil {
		t.Fatal("expected error for insufficient credits")
	}
}

func TestSelfRedemptionBlocked(t *testing.T) {
	k, ctx, bk, sk := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(types.ModuleName, "ulgm", sdkmath.NewInt(500000000))

	sk.setValidator(providerAddr1.String(), "bonded", "1111000000")

	regMsg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "1000",
		StakeAmount: "100000000",
	}
	_ = k.RegisterProvider(ctx, regMsg)

	k.GrantCredits(ctx, providerAddr1.String(), "claim-1", 0, 0)

	// Self-redemption should fail
	_, err := k.CreateRedemptionTicket(ctx, providerAddr1.String(), providerAddr1.String(), 50)
	if err == nil {
		t.Fatal("expected error for self-redemption")
	}
}

// -----------------------------------------------------------------------
// Tests: Pool Stats & Epoch
// -----------------------------------------------------------------------

func TestEpochStats(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Grant credits and verify epoch stats
	k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0) // epoch 0 (block 100)

	stats := k.GetOrCreateEpochStats(ctx, 0)
	if stats.CreditsIssued != 100 {
		t.Errorf("expected credits_issued=100, got %d", stats.CreditsIssued)
	}
}

func TestRecordPoolInflow(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.RecordPoolInflow(ctx, "1000000")
	k.RecordPoolInflow(ctx, "500000")

	stats := k.GetOrCreateEpochStats(ctx, 0)
	if stats.PoolInflow != "1500000" {
		t.Errorf("expected pool_inflow=1500000, got %s", stats.PoolInflow)
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis
// -----------------------------------------------------------------------

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set some state
	k.SetProvider(ctx, types.ComputeProvider{
		Address:     "addr1",
		ServiceType: "ai_inference",
		Status:      "active",
		Stake:       "100000000",
		PricePerCU:  "5000",
	})

	k.SetCredit(ctx, types.ComputeCredit{
		ValidatorAddr: "val1",
		Balance:       500,
		EarnedTotal:   1000,
	})

	// Export
	genState := k.ExportGenesis(ctx)

	if len(genState.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(genState.Providers))
	}
	if len(genState.Credits) != 1 {
		t.Fatalf("expected 1 credit, got %d", len(genState.Credits))
	}

	// Clear and reimport
	k2, ctx2, _, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, *genState)

	// Verify
	provider, found := k2.GetProvider(ctx2, "addr1")
	if !found {
		t.Fatal("provider not found after genesis import")
	}
	if provider.Status != "active" {
		t.Errorf("expected status=active, got %s", provider.Status)
	}

	credit, found := k2.GetCredit(ctx2, "val1")
	if !found {
		t.Fatal("credit not found after genesis import")
	}
	if credit.Balance != 500 {
		t.Errorf("expected balance=500, got %d", credit.Balance)
	}
}

// -----------------------------------------------------------------------
// Tests: Active Provider Queries
// -----------------------------------------------------------------------

func TestGetActiveProvidersByType(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(200000000))

	// Register two providers of different types
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api1.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	})

	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      providerAddr2.String(),
		ServiceType: "gpu_compute",
		Endpoint:    "https://api2.example.com",
		PricePerCU:  "3000",
		StakeAmount: "100000000",
	})

	// Query AI providers only
	aiProviders := k.GetActiveProvidersByType(ctx, "ai_inference")
	if len(aiProviders) != 1 {
		t.Errorf("expected 1 AI provider, got %d", len(aiProviders))
	}

	// Query all providers
	all := k.GetActiveProvidersByType(ctx, "")
	if len(all) != 2 {
		t.Errorf("expected 2 total providers, got %d", len(all))
	}
}

// -----------------------------------------------------------------------
// Tests: Service Type Validation
// -----------------------------------------------------------------------

func TestIsValidServiceType(t *testing.T) {
	if !types.IsValidServiceType("ai_inference") {
		t.Error("ai_inference should be valid")
	}
	if !types.IsValidServiceType("gpu_compute") {
		t.Error("gpu_compute should be valid")
	}
	if !types.IsValidServiceType("storage") {
		t.Error("storage should be valid")
	}
	if types.IsValidServiceType("quantum_teleport") {
		t.Error("quantum_teleport should be invalid")
	}
}

// -----------------------------------------------------------------------
// Tests: Message Validation
// -----------------------------------------------------------------------

func TestMsgRegisterValidateBasic(t *testing.T) {
	// Valid message
	msg := &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCU:  "5000",
		StakeAmount: "100000000",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	// Missing sender
	bad := *msg
	bad.Sender = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing sender")
	}

	// Invalid service type
	bad = *msg
	bad.ServiceType = "invalid"
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for invalid service type")
	}

	// Missing endpoint
	bad = *msg
	bad.Endpoint = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing endpoint")
	}

	// Zero stake
	bad = *msg
	bad.StakeAmount = "0"
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero stake")
	}
}

func TestMsgRedeemCreditsValidateBasic(t *testing.T) {
	msg := &types.MsgRedeemCredits{
		Sender:   validatorAddr.String(),
		Provider: providerAddr1.String(),
		CUAmount: 100,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	// Zero CU
	bad := *msg
	bad.CUAmount = 0
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero CU amount")
	}
}

// -----------------------------------------------------------------------
// Tests: Governance Auth
// -----------------------------------------------------------------------

func TestUpdateParamsAuth(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)

	// Wrong authority
	_, err := ms.UpdateComputeParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateComputeParams{
		Authority: "wrong_addr",
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected error for wrong authority")
	}
}

// -----------------------------------------------------------------------
// Tests: Utilization-Based Auto-Tuning
// -----------------------------------------------------------------------

func TestAdjustmentFactorDefault(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Default should be 1.0x = 1000000 bps
	factor := k.GetAdjustmentFactor(ctx)
	if factor != types.DefaultAdjustmentFactorBps {
		t.Errorf("expected default factor=%d, got %d", types.DefaultAdjustmentFactorBps, factor)
	}
}

func TestAdjustmentFactorSetGet(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetAdjustmentFactor(ctx, 1500000)
	factor := k.GetAdjustmentFactor(ctx)
	if factor != 1500000 {
		t.Errorf("expected factor=1500000, got %d", factor)
	}
}

func TestFinalizeEpochUnderutilized(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Epoch 0: inflow=1000000, outflow=50000 (5% utilization, well below 20% target low)
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "1000000",
		LGMDistributed: "50000", // 5% utilization
	}
	k.SetEpochStats(ctx, epoch0Stats)

	// Trigger epoch boundary at block 34272 (start of epoch 1 → finalizes epoch 0)
	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	// Factor should have increased (was 1000000, should go up)
	factor := k.GetAdjustmentFactor(ctx)
	if factor <= types.DefaultAdjustmentFactorBps {
		t.Errorf("expected factor > %d after underutilization, got %d", types.DefaultAdjustmentFactorBps, factor)
	}

	// Check epoch stats were updated
	stats, found := k.GetEpochStats(ctx, 0)
	if !found {
		t.Fatal("epoch 0 stats not found")
	}
	if stats.UtilizationBps != 50000 { // 5%
		t.Errorf("expected utilization_bps=50000, got %d", stats.UtilizationBps)
	}
	if stats.AdjustmentFactorBps != factor {
		t.Errorf("expected stats.adjustment_factor_bps=%d, got %d", factor, stats.AdjustmentFactorBps)
	}
}

func TestFinalizeEpochOverutilized(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Epoch 0: 95% utilization (above 80% target high)
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "1000000",
		LGMDistributed: "950000", // 95%
	}
	k.SetEpochStats(ctx, epoch0Stats)

	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	// Factor should have decreased
	factor := k.GetAdjustmentFactor(ctx)
	if factor >= types.DefaultAdjustmentFactorBps {
		t.Errorf("expected factor < %d after overutilization, got %d", types.DefaultAdjustmentFactorBps, factor)
	}
}

func TestFinalizeEpochSteadyState(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Epoch 0: 50% utilization (within 20%-80% target range)
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "1000000",
		LGMDistributed: "500000", // 50%
	}
	k.SetEpochStats(ctx, epoch0Stats)

	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	// Factor should be unchanged
	factor := k.GetAdjustmentFactor(ctx)
	if factor != types.DefaultAdjustmentFactorBps {
		t.Errorf("expected factor=%d in steady state, got %d", types.DefaultAdjustmentFactorBps, factor)
	}
}

func TestFinalizeEpochZeroInflow(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Epoch 0: zero inflow — factor should stay unchanged
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "0",
		LGMDistributed: "0",
	}
	k.SetEpochStats(ctx, epoch0Stats)

	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	factor := k.GetAdjustmentFactor(ctx)
	if factor != types.DefaultAdjustmentFactorBps {
		t.Errorf("expected no change with zero inflow, got factor=%d", factor)
	}
}

func TestAdjustmentFactorClampedAtMin(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set factor near the minimum
	k.SetAdjustmentFactor(ctx, types.DefaultMinAdjustmentFactorBps+1)

	// Epoch 0: 100% utilization — maximal pressure to decrease
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "1000000",
		LGMDistributed: "1000000",
	}
	k.SetEpochStats(ctx, epoch0Stats)

	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	factor := k.GetAdjustmentFactor(ctx)
	if factor < types.DefaultMinAdjustmentFactorBps {
		t.Errorf("factor %d should not go below min %d", factor, types.DefaultMinAdjustmentFactorBps)
	}
}

func TestAdjustmentFactorClampedAtMax(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set factor near the maximum
	k.SetAdjustmentFactor(ctx, types.DefaultMaxAdjustmentFactorBps-1)

	// Epoch 0: 0% utilization — maximal pressure to increase
	epoch0Stats := types.EpochStats{
		Epoch:          0,
		PoolInflow:     "1000000",
		LGMDistributed: "0",
	}
	k.SetEpochStats(ctx, epoch0Stats)

	ctx = ctx.WithBlockHeight(int64(types.BlocksPerEpoch))
	k.FinalizeEpoch(ctx)

	factor := k.GetAdjustmentFactor(ctx)
	if factor > types.DefaultMaxAdjustmentFactorBps {
		t.Errorf("factor %d should not exceed max %d", factor, types.DefaultMaxAdjustmentFactorBps)
	}
}

func TestGrantCreditsWithAdjustmentFactor(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Set adjustment factor to 2.0x (2000000 bps)
	k.SetAdjustmentFactor(ctx, 2000000)

	// Grant credits — base would be 100 CU for bonded with no complexity
	cuAmount, err := k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0)
	if err != nil {
		t.Fatalf("failed to grant: %v", err)
	}

	// 100 base × 2.0 adjustment = 200
	if cuAmount != 200 {
		t.Errorf("expected 200 CU with 2.0x factor, got %d", cuAmount)
	}
}

func TestGrantCreditsWithLowAdjustmentFactor(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "bonded", "1111000000")

	// Set adjustment factor to 0.5x (500000 bps)
	k.SetAdjustmentFactor(ctx, 500000)

	cuAmount, err := k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0)
	if err != nil {
		t.Fatalf("failed to grant: %v", err)
	}

	// 100 base × 0.5 adjustment = 50
	if cuAmount != 50 {
		t.Errorf("expected 50 CU with 0.5x factor, got %d", cuAmount)
	}
}

func TestGrantCreditsMinimumOneCU(t *testing.T) {
	k, ctx, _, sk := setupKeeper(t)

	sk.setValidator(validatorAddr.String(), "apprentice", "111000000")

	// Set factor to minimum (0.2x)
	k.SetAdjustmentFactor(ctx, 200000)

	// Apprentice with no complexity: base = 50 CU × 0.2 = 10
	cuAmount, err := k.GrantCredits(ctx, validatorAddr.String(), "claim-1", 0, 0)
	if err != nil {
		t.Fatalf("failed to grant: %v", err)
	}

	// 50 × 200000 / 1000000 = 10
	if cuAmount != 10 {
		t.Errorf("expected 10 CU, got %d", cuAmount)
	}
}

func TestMultiEpochAdjustmentConvergence(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Simulate 5 epochs of underutilization — factor should steadily increase
	for epoch := uint64(0); epoch < 5; epoch++ {
		stats := types.EpochStats{
			Epoch:          epoch,
			PoolInflow:     "1000000",
			LGMDistributed: "100000", // 10% utilization
		}
		k.SetEpochStats(ctx, stats)

		nextEpochBlock := int64((epoch + 1) * types.BlocksPerEpoch)
		ctx = ctx.WithBlockHeight(nextEpochBlock)
		k.FinalizeEpoch(ctx)
	}

	factor := k.GetAdjustmentFactor(ctx)
	// After 5 epochs of underutilization, factor should be well above 1.0x
	if factor <= types.DefaultAdjustmentFactorBps {
		t.Errorf("expected factor > %d after 5 underutilized epochs, got %d", types.DefaultAdjustmentFactorBps, factor)
	}

	// Now simulate 5 epochs of overutilization — should decrease
	startFactor := factor
	for epoch := uint64(5); epoch < 10; epoch++ {
		stats := types.EpochStats{
			Epoch:          epoch,
			PoolInflow:     "1000000",
			LGMDistributed: "950000", // 95% utilization
		}
		k.SetEpochStats(ctx, stats)

		nextEpochBlock := int64((epoch + 1) * types.BlocksPerEpoch)
		ctx = ctx.WithBlockHeight(nextEpochBlock)
		k.FinalizeEpoch(ctx)
	}

	factor = k.GetAdjustmentFactor(ctx)
	if factor >= startFactor {
		t.Errorf("expected factor to decrease after overutilization, was %d now %d", startFactor, factor)
	}
}

func TestAutoTuningParamsValidation(t *testing.T) {
	// Low >= High should fail
	params := types.DefaultParams()
	params.TargetUtilizationLowBps = 800000
	params.TargetUtilizationHighBps = 200000
	if err := params.Validate(); err == nil {
		t.Error("expected error for low >= high")
	}

	// High > 1000000 should fail
	params = types.DefaultParams()
	params.TargetUtilizationHighBps = 1000001
	if err := params.Validate(); err == nil {
		t.Error("expected error for high > 1000000")
	}

	// Zero max adjustment should fail
	params = types.DefaultParams()
	params.MaxAdjustmentPerEpochBps = 0
	if err := params.Validate(); err == nil {
		t.Error("expected error for zero max adjustment")
	}

	// Zero min factor should fail
	params = types.DefaultParams()
	params.MinAdjustmentFactorBps = 0
	if err := params.Validate(); err == nil {
		t.Error("expected error for zero min factor")
	}

	// Min >= Max factor should fail
	params = types.DefaultParams()
	params.MinAdjustmentFactorBps = 5000000
	params.MaxAdjustmentFactorBps = 200000
	if err := params.Validate(); err == nil {
		t.Error("expected error for min >= max factor")
	}
}
