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

	"github.com/legible-money/legible/x/partnerships/keeper"
	"github.com/legible-money/legible/x/partnerships/types"
)

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
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

// -----------------------------------------------------------------------
// Mock StakingKeeper
// -----------------------------------------------------------------------

type mockStakingKeeper struct{}

func (m *mockStakingKeeper) GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (int64, error) {
	return 100, nil
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
	sk := &mockStakingKeeper{}

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, sk, authority)

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
	if params.FormationWindowBlocks != 77 {
		t.Errorf("expected FormationWindowBlocks 77, got %d", params.FormationWindowBlocks)
	}
	if params.CommonPotShareBps != 220000 {
		t.Errorf("expected CommonPotShareBps 220000, got %d", params.CommonPotShareBps)
	}
	if params.DefaultHumanSplitBps != 500000 {
		t.Errorf("expected DefaultHumanSplitBps 500000, got %d", params.DefaultHumanSplitBps)
	}
	if params.DefaultAgentSplitBps != 500000 {
		t.Errorf("expected DefaultAgentSplitBps 500000, got %d", params.DefaultAgentSplitBps)
	}
}

func TestParamsSetGet(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	custom := types.Params{
		FormationWindowBlocks:      100,
		CoolingPeriodBlocks:        500000,
		CommonPotShareBps:          300000,
		SafetyFreezeDurationBlocks: 500,
		MaxFreezesPerEpoch:         5,
		CoercionReviewBlocks:       2000,
		BaseCooldownBlocks:         100,
		MaxCounterProposalDepth:    5,
		DefaultHumanSplitBps:       600000,
		DefaultAgentSplitBps:       400000,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.FormationWindowBlocks != 100 {
		t.Errorf("expected 100, got %d", got.FormationWindowBlocks)
	}
	if got.CommonPotShareBps != 300000 {
		t.Errorf("expected 300000, got %d", got.CommonPotShareBps)
	}
}

func TestParamsValidation(t *testing.T) {
	cases := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{"valid", types.DefaultParams(), false},
		{"zero formation window", types.Params{FormationWindowBlocks: 0, CoolingPeriodBlocks: 222222, CommonPotShareBps: 220000, SafetyFreezeDurationBlocks: 222, MaxFreezesPerEpoch: 3, CoercionReviewBlocks: 1111, BaseCooldownBlocks: 77, MaxCounterProposalDepth: 3, DefaultHumanSplitBps: 500000, DefaultAgentSplitBps: 500000}, true},
		{"splits dont sum to 1000000", types.Params{FormationWindowBlocks: 77, CoolingPeriodBlocks: 222222, CommonPotShareBps: 220000, SafetyFreezeDurationBlocks: 222, MaxFreezesPerEpoch: 3, CoercionReviewBlocks: 1111, BaseCooldownBlocks: 77, MaxCounterProposalDepth: 3, DefaultHumanSplitBps: 600000, DefaultAgentSplitBps: 500000}, true},
		{"zero common pot", types.Params{FormationWindowBlocks: 77, CoolingPeriodBlocks: 222222, CommonPotShareBps: 0, SafetyFreezeDurationBlocks: 222, MaxFreezesPerEpoch: 3, CoercionReviewBlocks: 1111, BaseCooldownBlocks: 77, MaxCounterProposalDepth: 3, DefaultHumanSplitBps: 500000, DefaultAgentSplitBps: 500000}, true},
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
// Tests: Partnership CRUD
// -----------------------------------------------------------------------

func TestPartnershipCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Not found initially
	_, found := k.GetPartnership(ctx, "p-1")
	if found {
		t.Error("expected partnership not found")
	}

	// Set
	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		Tier:             types.TierTrial,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
		FormedAtBlock:    100,
	}
	k.SetPartnership(ctx, p)

	// Get
	got, found := k.GetPartnership(ctx, "p-1")
	if !found {
		t.Fatal("expected partnership to be found")
	}
	if got.HumanAddr != testAddr(1) {
		t.Errorf("expected human addr %s, got %s", testAddr(1), got.HumanAddr)
	}
	if got.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", got.Status)
	}

	// By Human index
	ids := k.GetPartnershipsByHuman(ctx, testAddr(1))
	if len(ids) != 1 || ids[0] != "p-1" {
		t.Errorf("expected 1 partnership by human, got %v", ids)
	}

	// By Agent index
	ids = k.GetPartnershipsByAgent(ctx, testAddr(2))
	if len(ids) != 1 || ids[0] != "p-1" {
		t.Errorf("expected 1 partnership by agent, got %v", ids)
	}

	// GetAll
	all := k.GetAllPartnerships(ctx)
	if len(all) != 1 {
		t.Errorf("expected 1 partnership, got %d", len(all))
	}

	// Delete
	k.DeletePartnership(ctx, p)
	_, found = k.GetPartnership(ctx, "p-1")
	if found {
		t.Error("expected partnership to be deleted")
	}
}

func TestPartnershipsByParticipant(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	// Human should find it
	result := k.GetPartnershipsByParticipant(ctx, human)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}

	// Agent should find it
	result = k.GetPartnershipsByParticipant(ctx, agent)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}

	// Non-participant
	result = k.GetPartnershipsByParticipant(ctx, testAddr(99))
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

// -----------------------------------------------------------------------
// Tests: Reward Distribution
// -----------------------------------------------------------------------

func TestDistributeReward(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		Tier:             types.TierTrial, // 1.0x multiplier
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Distribute 10000 with Trial tier (1.0x multiplier)
	humanShare, agentShare, commonPotAdd, err := k.DistributeReward(ctx, &p, "10000", "verification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With Trial tier (1.0x), adjusted = 10000
	// Common pot = 10000 * 220000 / 1000000 = 2200
	// Remaining = 7800
	// Human = 7800 * 500000 / 1000000 = 3900
	// Agent = 7800 - 3900 = 3900
	if commonPotAdd != "2200" {
		t.Errorf("expected common pot 2200, got %s", commonPotAdd)
	}
	if humanShare != "3900" {
		t.Errorf("expected human share 3900, got %s", humanShare)
	}
	if agentShare != "3900" {
		t.Errorf("expected agent share 3900, got %s", agentShare)
	}

	// Verify partnership metrics updated
	updated, _ := k.GetPartnership(ctx, "p-1")
	if updated.CommonPotBalance != "2200" {
		t.Errorf("expected pot balance 2200, got %s", updated.CommonPotBalance)
	}
	if updated.TotalEarned != "10000" {
		t.Errorf("expected total earned 10000, got %s", updated.TotalEarned)
	}
}

func TestDistributeRewardEliteTier(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-2",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		Tier:             types.TierElite,
		LockTier:         3, // 1.44x multiplier (LockTiers[3])
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	humanShare, agentShare, commonPotAdd, err := k.DistributeReward(ctx, &p, "10000", "citation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LockTier 3: adjusted = 10000 * 1440000 / 1000000 = 14400
	// Common pot = 14400 * 220000 / 1000000 = 3168
	// Remaining = 14400 - 3168 = 11232
	// Human = 11232 * 500000 / 1000000 = 5616
	// Agent = 11232 - 5616 = 5616
	if commonPotAdd != "3168" {
		t.Errorf("expected common pot 3168, got %s", commonPotAdd)
	}
	if humanShare != "5616" {
		t.Errorf("expected human share 5616, got %s", humanShare)
	}
	if agentShare != "5616" {
		t.Errorf("expected agent share 5616, got %s", agentShare)
	}
}

func TestDistributeRewardAccumulates(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-3",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		Tier:             types.TierTrial,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "1000",
		TotalEarned:      "5000",
	}
	k.SetPartnership(ctx, p)

	_, _, potAdd, err := k.DistributeReward(ctx, &p, "10000", "service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := k.GetPartnership(ctx, "p-3")
	// Previous pot 1000 + new 2200 = 3200
	expectedPot := new(big.Int).SetInt64(1000)
	addedPot := new(big.Int)
	addedPot.SetString(potAdd, 10)
	expectedPot.Add(expectedPot, addedPot)

	if updated.CommonPotBalance != expectedPot.String() {
		t.Errorf("expected pot %s, got %s", expectedPot.String(), updated.CommonPotBalance)
	}
	// Previous earned 5000 + 10000 = 15000
	if updated.TotalEarned != "15000" {
		t.Errorf("expected total earned 15000, got %s", updated.TotalEarned)
	}
}

// -----------------------------------------------------------------------
// Tests: Deliberation
// -----------------------------------------------------------------------

func TestCalculateAmountTier(t *testing.T) {
	k, _, _ := setupKeeper(t)

	cases := []struct {
		name       string
		amount     string
		potBalance string
		expected   types.AmountTier
	}{
		{"micro (<1%)", "50", "10000", types.AmountTierMicro},
		{"small (1-5%)", "300", "10000", types.AmountTierSmall},
		{"medium (5-22%)", "1000", "10000", types.AmountTierMedium},
		{"large (22-55%)", "3000", "10000", types.AmountTierLarge},
		{"major (>55%)", "6000", "10000", types.AmountTierMajor},
		{"empty pot is major", "100", "0", types.AmountTierMajor},
		{"zero amount is micro", "0", "10000", types.AmountTierMicro},
		{"exact 1% boundary", "100", "10000", types.AmountTierSmall},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tier := k.CalculateAmountTier(tc.amount, tc.potBalance)
			if tier != tc.expected {
				t.Errorf("expected tier %s, got %s", tc.expected, tier)
			}
		})
	}
}

func TestCalculateCooldown(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // BaseCooldownBlocks = 77

	cases := []struct {
		rejections uint32
		expected   uint64
	}{
		{0, 77},     // 77 * 2^0 = 77
		{1, 154},    // 77 * 2^1 = 154
		{2, 308},    // 77 * 2^2 = 308
		{3, 616},    // 77 * 2^3 = 616
		{7, 9856},   // 77 * 2^7 = 9856
		{8, 11111},  // 77 * 2^8 = 19712 > 11111, capped
		{20, 11111}, // capped
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d rejections", tc.rejections), func(t *testing.T) {
			cooldown := k.CalculateCooldown(ctx, tc.rejections)
			if cooldown != tc.expected {
				t.Errorf("expected cooldown %d, got %d", tc.expected, cooldown)
			}
		})
	}
}

func TestValidateCounterProposal(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // MaxCounterProposalDepth = 3

	// Depth 2 should be OK (can create depth 3)
	op := types.ConsensusOperation{
		Deliberation: &types.DeliberationState{ChainDepth: 2},
	}
	if err := k.ValidateCounterProposal(ctx, op); err != nil {
		t.Errorf("expected no error at depth 2, got: %v", err)
	}

	// Depth 3 should fail (cannot create depth 4)
	op.Deliberation.ChainDepth = 3
	if err := k.ValidateCounterProposal(ctx, op); err == nil {
		t.Error("expected error at depth 3")
	}
}

// -----------------------------------------------------------------------
// Tests: Safety Freeze
// -----------------------------------------------------------------------

func TestSafetyFreeze(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Apply first freeze
	sf, err := k.HandleSafetyFreeze(ctx, "p-1", testAddr(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.ExpiresAt != 322 { // 100 + 222
		t.Errorf("expected expires at 322, got %d", sf.ExpiresAt)
	}
	if sf.FreezeCountThisEpoch != 1 {
		t.Errorf("expected freeze count 1, got %d", sf.FreezeCountThisEpoch)
	}

	// Partnership should be suspended
	updated, _ := k.GetPartnership(ctx, "p-1")
	if updated.Status != types.StatusSuspended {
		t.Errorf("expected suspended, got %s", updated.Status)
	}

	// Cooperation score should be penalized
	if updated.CooperationScore != 4900 { // 5000 - 100*1
		t.Errorf("expected cooperation 4900, got %d", updated.CooperationScore)
	}
}

func TestSafetyFreezeMaxReached(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Set freeze count to max
	sf := types.SafetyFreeze{
		PartnershipId:        "p-1",
		FreezeCountThisEpoch: 3,
		ExpiresAt:            50, // already expired
	}
	k.SetSafetyFreeze(ctx, sf)

	_, err := k.HandleSafetyFreeze(ctx, "p-1", testAddr(1))
	if err == nil {
		t.Error("expected max freezes error")
	}
}

func TestSafetyFreezeAlreadyActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Set active freeze
	sf := types.SafetyFreeze{
		PartnershipId:        "p-1",
		FreezeCountThisEpoch: 1,
		ExpiresAt:            200, // still active (current=100)
	}
	k.SetSafetyFreeze(ctx, sf)

	_, err := k.HandleSafetyFreeze(ctx, "p-1", testAddr(1))
	if err == nil {
		t.Error("expected freeze active error")
	}
}

// -----------------------------------------------------------------------
// Tests: Coercion Signal
// -----------------------------------------------------------------------

func TestCoercionSignal(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	cs, err := k.HandleCoercionSignal(ctx, "p-1", testAddr(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.ExpiresAt != 1211 { // 100 + 1111
		t.Errorf("expected expires at 1211, got %d", cs.ExpiresAt)
	}
	if cs.Resolved {
		t.Error("expected not resolved")
	}

	// Partnership should be suspended
	updated, _ := k.GetPartnership(ctx, "p-1")
	if updated.Status != types.StatusSuspended {
		t.Errorf("expected suspended, got %s", updated.Status)
	}
}

func TestCoercionSignalAlreadyActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	// First signal
	_, err := k.HandleCoercionSignal(ctx, "p-1", testAddr(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second signal should fail
	_, err = k.HandleCoercionSignal(ctx, "p-1", testAddr(2))
	if err == nil {
		t.Error("expected coercion active error")
	}
}

// -----------------------------------------------------------------------
// Tests: Exit
// -----------------------------------------------------------------------

func TestHandleExit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "50000",
	}
	k.SetPartnership(ctx, p)

	cooldownEnd, err := k.HandleExit(ctx, "p-1", testAddr(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cooldown = 100 + 222222 = 222322
	if cooldownEnd != 222322 {
		t.Errorf("expected cooldown end 222322, got %d", cooldownEnd)
	}

	// Partnership should be cooling
	updated, _ := k.GetPartnership(ctx, "p-1")
	if updated.Status != types.StatusCooling {
		t.Errorf("expected cooling, got %s", updated.Status)
	}
	if updated.ExitState == nil {
		t.Fatal("expected exit state to be set")
	}
	if updated.ExitState.InitiatedBy != testAddr(1) {
		t.Errorf("expected initiator %s, got %s", testAddr(1), updated.ExitState.InitiatedBy)
	}
	if updated.CommonPotBalance != "0" {
		t.Errorf("expected pot cleared to 0, got %s", updated.CommonPotBalance)
	}
}

func TestHandleExitAlreadyDissolved(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusDissolved,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	_, err := k.HandleExit(ctx, "p-1", testAddr(1))
	if err == nil {
		t.Error("expected already dissolved error")
	}
}

// -----------------------------------------------------------------------
// Tests: MsgServer
// -----------------------------------------------------------------------

func TestMsgInitiateAndAcceptPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	// Initiate
	resp, err := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}
	if resp.PartnershipId == "" {
		t.Fatal("expected non-empty partnership ID")
	}

	// Verify forming status
	p, found := k.GetPartnership(ctx, resp.PartnershipId)
	if !found {
		t.Fatal("partnership not found")
	}
	if p.Status != types.StatusForming {
		t.Errorf("expected forming, got %s", p.Status)
	}

	// Accept
	_, err = srv.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		Acceptor:      agent,
		PartnershipId: resp.PartnershipId,
	})
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}

	// Verify active status
	p, _ = k.GetPartnership(ctx, resp.PartnershipId)
	if p.Status != types.StatusActive {
		t.Errorf("expected active, got %s", p.Status)
	}
}

func TestMsgAcceptPartnershipWrongAcceptor(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: testAddr(1),
		Partner:   testAddr(2),
	})

	// Try to accept as wrong person
	_, err := srv.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		Acceptor:      testAddr(3),
		PartnershipId: resp.PartnershipId,
	})
	if err == nil {
		t.Error("expected error for wrong acceptor")
	}
}

func TestMsgDistributeReward(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	authority := k.GetAuthority()

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		Tier:             types.TierStandard,
		LockTier:         1, // 1.11x multiplier (LockTiers[1])
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.DistributeReward(ctx, &types.MsgDistributeReward{
		Authority:     authority,
		PartnershipId: "p-1",
		GrossAmount:   "10000",
		SourceType:    "verification",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LockTier 1 (1.11x): adjusted = 10000 * 1110000 / 1000000 = 11100
	// Common pot = 11100 * 220000 / 1000000 = 2442
	// Remaining = 11100 - 2442 = 8658
	// Human = 8658 * 500000 / 1000000 = 4329
	// Agent = 8658 - 4329 = 4329
	if resp.CommonPotAdd != "2442" {
		t.Errorf("expected common pot 2442, got %s", resp.CommonPotAdd)
	}
	if resp.HumanShare != "4329" {
		t.Errorf("expected human 4329, got %s", resp.HumanShare)
	}
	if resp.AgentShare != "4329" {
		t.Errorf("expected agent 4329, got %s", resp.AgentShare)
	}
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	newParams := types.Params{
		FormationWindowBlocks:      100,
		CoolingPeriodBlocks:        300000,
		CommonPotShareBps:          250000,
		SafetyFreezeDurationBlocks: 300,
		MaxFreezesPerEpoch:         5,
		CoercionReviewBlocks:       2000,
		BaseCooldownBlocks:         100,
		MaxCounterProposalDepth:    5,
		DefaultHumanSplitBps:       600000,
		DefaultAgentSplitBps:       400000,
	}
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.CommonPotShareBps != 250000 {
		t.Errorf("expected 250000, got %d", got.CommonPotShareBps)
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

func TestMsgSafetyFreeze(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.SafetyFreeze(ctx, &types.MsgSafetyFreeze{
		Freezer:       testAddr(1),
		PartnershipId: "p-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ExpiresAt != 322 {
		t.Errorf("expected 322, got %d", resp.ExpiresAt)
	}
}

func TestMsgSafetyFreezeNotParticipant(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SafetyFreeze(ctx, &types.MsgSafetyFreeze{
		Freezer:       testAddr(99),
		PartnershipId: "p-1",
	})
	if err == nil {
		t.Error("expected not participant error")
	}
}

func TestMsgInitiateExit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "5000",
		TotalEarned:      "20000",
	}
	k.SetPartnership(ctx, p)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.InitiateExit(ctx, &types.MsgInitiateExit{
		Initiator:     testAddr(1),
		PartnershipId: "p-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CooldownEndsAt != 222322 { // 100 + 222222
		t.Errorf("expected 222322, got %d", resp.CooldownEndsAt)
	}
}

// -----------------------------------------------------------------------
// Tests: Queries
// -----------------------------------------------------------------------

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.FormationWindowBlocks != 77 {
		t.Errorf("expected 77, got %d", resp.Params.FormationWindowBlocks)
	}
}

func TestQueryPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "1000",
		TotalEarned:      "5000",
	}
	k.SetPartnership(ctx, p)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Partnership(ctx, &types.QueryPartnershipRequest{PartnershipId: "p-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Partnership.CommonPotBalance != "1000" {
		t.Errorf("expected 1000, got %s", resp.Partnership.CommonPotBalance)
	}
}

func TestQueryPartnershipNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Partnership(ctx, &types.QueryPartnershipRequest{PartnershipId: "nonexistent"})
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestQueryPartnershipsByParticipant(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(1)
	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        human,
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.PartnershipsByParticipant(ctx, &types.QueryPartnershipsByParticipantRequest{Participant: human})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Partnerships) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Partnerships))
	}
}

func TestQueryPartnershipNilRequest(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Partnership(ctx, nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis
// -----------------------------------------------------------------------

func TestGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-1",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "5000",
		TotalEarned:      "20000",
		CooperationScore: 7500,
		FormedAtBlock:    50,
	}
	k.SetPartnership(ctx, p)

	// Export
	genState := k.ExportGenesis(ctx)
	if len(genState.Partnerships) != 1 {
		t.Errorf("expected 1 partnership, got %d", len(genState.Partnerships))
	}

	// Import into new keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, *genState)

	got := k2.ExportGenesis(ctx2)
	if len(got.Partnerships) != 1 {
		t.Errorf("expected 1 partnership after import, got %d", len(got.Partnerships))
	}
	if got.Partnerships[0].CommonPotBalance != "5000" {
		t.Errorf("expected pot 5000, got %s", got.Partnerships[0].CommonPotBalance)
	}
}

func TestGenesisValidation(t *testing.T) {
	valid := types.DefaultGenesis()
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Duplicate ID
	dup := types.DefaultGenesis()
	dup.Partnerships = []*types.Partnership{
		{Id: "p-1", HumanAddr: "a", AgentAddr: "b", SplitHumanBps: 500000, SplitAgentBps: 500000},
		{Id: "p-1", HumanAddr: "c", AgentAddr: "d", SplitHumanBps: 500000, SplitAgentBps: 500000},
	}
	if err := dup.Validate(); err == nil {
		t.Error("expected duplicate ID error")
	}

	// Invalid splits
	badSplits := types.DefaultGenesis()
	badSplits.Partnerships = []*types.Partnership{
		{Id: "p-2", HumanAddr: "a", AgentAddr: "b", SplitHumanBps: 600000, SplitAgentBps: 500000},
	}
	if err := badSplits.Validate(); err == nil {
		t.Error("expected invalid splits error")
	}
}

// -----------------------------------------------------------------------
// Tests: ValidateBasic
// -----------------------------------------------------------------------

func TestMsgInitiatePartnershipValidateBasic(t *testing.T) {
	human := testAddr(1)
	agent := testAddr(2)

	cases := []struct {
		name    string
		msg     types.MsgInitiatePartnership
		wantErr bool
	}{
		{"valid", types.MsgInitiatePartnership{Initiator: human, Partner: agent}, false},
		{"empty initiator", types.MsgInitiatePartnership{Initiator: "", Partner: agent}, true},
		{"empty partner", types.MsgInitiatePartnership{Initiator: human, Partner: ""}, true},
		{"self partnership", types.MsgInitiatePartnership{Initiator: human, Partner: human}, true},
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

func TestMsgDistributeRewardValidateBasic(t *testing.T) {
	authority := testAddr(1)

	cases := []struct {
		name    string
		msg     types.MsgDistributeReward
		wantErr bool
	}{
		{"valid", types.MsgDistributeReward{Authority: authority, PartnershipId: "p-1", GrossAmount: "1000", SourceType: "verification"}, false},
		{"empty authority", types.MsgDistributeReward{Authority: "", PartnershipId: "p-1", GrossAmount: "1000", SourceType: "verification"}, true},
		{"empty partnership", types.MsgDistributeReward{Authority: authority, PartnershipId: "", GrossAmount: "1000", SourceType: "verification"}, true},
		{"zero amount", types.MsgDistributeReward{Authority: authority, PartnershipId: "p-1", GrossAmount: "0", SourceType: "verification"}, true},
		{"invalid amount", types.MsgDistributeReward{Authority: authority, PartnershipId: "p-1", GrossAmount: "abc", SourceType: "verification"}, true},
		{"empty source", types.MsgDistributeReward{Authority: authority, PartnershipId: "p-1", GrossAmount: "1000", SourceType: ""}, true},
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
		Params:    types.Params{FormationWindowBlocks: 0},
	}
	if err := invalidParams.ValidateBasic(); err == nil {
		t.Error("expected error for invalid params")
	}
}

// -----------------------------------------------------------------------
// Tests: Sequence
// -----------------------------------------------------------------------

func TestSequence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	seq1 := k.NextSequence(ctx)
	seq2 := k.NextSequence(ctx)
	seq3 := k.NextSequence(ctx)

	if seq1 != 1 {
		t.Errorf("expected 1, got %d", seq1)
	}
	if seq2 != 2 {
		t.Errorf("expected 2, got %d", seq2)
	}
	if seq3 != 3 {
		t.Errorf("expected 3, got %d", seq3)
	}
}

// -----------------------------------------------------------------------
// Tests: ConsensusOperation CRUD
// -----------------------------------------------------------------------

// -----------------------------------------------------------------------
// Tests: Lock Tier System
// -----------------------------------------------------------------------

func TestLockTierAssignment(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	cases := []struct {
		name            string
		lockTier        uint32
		wantLockTier    uint32
		wantLockExpires uint64
	}{
		{"tier 0", 0, 0, 100 + 22222},
		{"tier 1", 1, 1, 100 + 77777},
		{"tier 2", 2, 2, 100 + 222222},
		{"tier 3", 3, 3, 100 + 777777},
		{"tier 4", 4, 4, 100 + 1111111},
		{"tier 5", 5, 5, 100 + 2222222},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			human := testAddr(i*2 + 10)
			agent := testAddr(i*2 + 11)

			resp, err := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
				Initiator: human,
				Partner:   agent,
				LockTier:  tc.lockTier,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			p, found := k.GetPartnership(ctx, resp.PartnershipId)
			if !found {
				t.Fatal("partnership not found")
			}
			if p.LockTier != tc.wantLockTier {
				t.Errorf("expected LockTier %d, got %d", tc.wantLockTier, p.LockTier)
			}
			if p.LockExpiresAt != tc.wantLockExpires {
				t.Errorf("expected LockExpiresAt %d, got %d", tc.wantLockExpires, p.LockExpiresAt)
			}
		})
	}
}

func TestLockTierInvalid(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	_, err := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: testAddr(1),
		Partner:   testAddr(2),
		LockTier:  6, // invalid: max is 5
	})
	if err == nil {
		t.Error("expected error for invalid lock tier")
	}
}

func TestExitBlockedByUnexpiredLock(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	// Partnership with lock expiring far in the future
	p := types.Partnership{
		Id:               "p-lock",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		LockTier:         2,
		LockExpiresAt:    999999, // far in the future (current block = 100)
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "5000",
		TotalEarned:      "10000",
	}
	k.SetPartnership(ctx, p)

	_, err := k.HandleExit(ctx, "p-lock", testAddr(1))
	if err == nil {
		t.Error("expected ErrLockNotExpired")
	}
}

func TestExitAllowedAfterLockExpires(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	// Partnership with lock already expired
	p := types.Partnership{
		Id:               "p-expired",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		LockTier:         1,
		LockExpiresAt:    50, // expired (current block = 100)
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "20000",
	}
	k.SetPartnership(ctx, p)

	cooldownEnd, err := k.HandleExit(ctx, "p-expired", testAddr(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Cooldown = 100 + 222222 = 222322
	if cooldownEnd != 222322 {
		t.Errorf("expected cooldown end 222322, got %d", cooldownEnd)
	}
}

func TestExitAllowedDuringFreeze(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	// Suspended (frozen) partnership with unexpired lock
	p := types.Partnership{
		Id:               "p-frozen",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusSuspended,
		LockTier:         3,
		LockExpiresAt:    999999, // lock not expired, but frozen overrides
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "8000",
		TotalEarned:      "30000",
	}
	k.SetPartnership(ctx, p)

	_, err := k.HandleExit(ctx, "p-frozen", testAddr(1))
	if err != nil {
		t.Fatalf("expected exit during freeze to succeed, got: %v", err)
	}

	updated, _ := k.GetPartnership(ctx, "p-frozen")
	if updated.Status != types.StatusCooling {
		t.Errorf("expected cooling, got %s", updated.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: Exit Settlement Math
// -----------------------------------------------------------------------

func TestCalculateExitSettlement_AllTiers(t *testing.T) {
	potBalance := big.NewInt(100000)

	cases := []struct {
		tier            uint32
		initiatorIsHuman bool
		wantHuman       string
		wantAgent       string
		wantBurned      string
	}{
		// Tier 0: 11% penalty, initiator=77%, non-initiator=55%
		// halfPot = 50000 each
		// tierPenalty = 50000 * 110000 / 1000000 = 5500
		// initiatorForfeited = 5500 * 7700 / 10000 = 4235
		// nonInitiatorForfeited = 5500 * 5500 / 10000 = 3025
		// initiatorPay = 50000 - 4235 = 45765
		// nonInitiatorPay = 50000 - 3025 = 46975
		{0, true, "45765", "46975", "7260"},

		// Tier 1: 22% penalty
		// tierPenalty = 50000 * 220000 / 1000000 = 11000
		// initiatorForfeited = 11000 * 7700 / 10000 = 8470
		// nonInitiatorForfeited = 11000 * 5500 / 10000 = 6050
		// initiatorPay = 50000 - 8470 = 41530
		// nonInitiatorPay = 50000 - 6050 = 43950
		{1, true, "41530", "43950", "14520"},

		// Tier 2: 33% penalty
		// tierPenalty = 50000 * 330000 / 1000000 = 16500
		// initiatorForfeited = 16500 * 7700 / 10000 = 12705
		// nonInitiatorForfeited = 16500 * 5500 / 10000 = 9075
		// initiatorPay = 50000 - 12705 = 37295
		// nonInitiatorPay = 50000 - 9075 = 40925
		{2, true, "37295", "40925", "21780"},

		// Tier 3: 44% penalty
		// tierPenalty = 50000 * 440000 / 1000000 = 22000
		// initiatorForfeited = 22000 * 7700 / 10000 = 16940
		// nonInitiatorForfeited = 22000 * 5500 / 10000 = 12100
		{3, true, "33060", "37900", "29040"},

		// Tier 4: 55% penalty
		// tierPenalty = 50000 * 550000 / 1000000 = 27500
		// initiatorForfeited = 27500 * 7700 / 10000 = 21175
		// nonInitiatorForfeited = 27500 * 5500 / 10000 = 15125
		{4, true, "28825", "34875", "36300"},

		// Tier 5: 66% penalty
		// tierPenalty = 50000 * 660000 / 1000000 = 33000
		// initiatorForfeited = 33000 * 7700 / 10000 = 25410
		// nonInitiatorForfeited = 33000 * 5500 / 10000 = 18150
		{5, true, "24590", "31850", "43560"},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("tier_%d", tc.tier)
		t.Run(name, func(t *testing.T) {
			humanAddr := "human"
			initiator := humanAddr
			if !tc.initiatorIsHuman {
				initiator = "agent"
			}

			humanPay, agentPay, burned := keeper.CalculateExitSettlement(
				new(big.Int).Set(potBalance), tc.tier, initiator, humanAddr,
			)

			if humanPay.String() != tc.wantHuman {
				t.Errorf("humanPayout: expected %s, got %s", tc.wantHuman, humanPay.String())
			}
			if agentPay.String() != tc.wantAgent {
				t.Errorf("agentPayout: expected %s, got %s", tc.wantAgent, agentPay.String())
			}
			if burned.String() != tc.wantBurned {
				t.Errorf("burned: expected %s, got %s", tc.wantBurned, burned.String())
			}

			// Conservation check: human + agent + burned == potBalance
			total := new(big.Int).Add(humanPay, agentPay)
			total.Add(total, burned)
			if total.Cmp(potBalance) != 0 {
				t.Errorf("conservation violated: %s + %s + %s = %s, want %s",
					humanPay, agentPay, burned, total, potBalance)
			}
		})
	}
}

func TestCalculateExitSettlement_AgentInitiator(t *testing.T) {
	potBalance := big.NewInt(100000)

	// Tier 1, agent initiates exit
	// Agent (initiator) should get less than human (non-initiator)
	humanPay, agentPay, burned := keeper.CalculateExitSettlement(
		potBalance, 1, "agent", "human",
	)

	// Agent is initiator: pays 77% of tier penalty
	// Human is non-initiator: pays 55% of tier penalty
	// So human should get more
	if humanPay.Cmp(agentPay) <= 0 {
		t.Errorf("non-initiator (human=%s) should get more than initiator (agent=%s)",
			humanPay, agentPay)
	}

	// Verify exact values (same math as tier 1 but swapped)
	if humanPay.String() != "43950" {
		t.Errorf("expected human 43950, got %s", humanPay)
	}
	if agentPay.String() != "41530" {
		t.Errorf("expected agent 41530, got %s", agentPay)
	}
	if burned.String() != "14520" {
		t.Errorf("expected burned 14520, got %s", burned)
	}
}

func TestCalculateExitSettlement_ZeroPot(t *testing.T) {
	humanPay, agentPay, burned := keeper.CalculateExitSettlement(
		big.NewInt(0), 3, "human", "human",
	)
	if humanPay.Sign() != 0 || agentPay.Sign() != 0 || burned.Sign() != 0 {
		t.Errorf("expected all zeros for empty pot, got h=%s a=%s b=%s",
			humanPay, agentPay, burned)
	}
}

// -----------------------------------------------------------------------
// Tests: MsgDepositToPot
// -----------------------------------------------------------------------

func TestMsgDepositToPot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-dep",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "1000",
		TotalEarned:      "5000",
	}
	k.SetPartnership(ctx, p)

	// Human deposits
	_, err := srv.DepositToPot(ctx, &types.MsgDepositToPot{
		Depositor:     human,
		PartnershipId: "p-dep",
		Amount:        "5000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := k.GetPartnership(ctx, "p-dep")
	if updated.CommonPotBalance != "6000" { // 1000 + 5000
		t.Errorf("expected pot 6000, got %s", updated.CommonPotBalance)
	}

	// Agent deposits
	_, err = srv.DepositToPot(ctx, &types.MsgDepositToPot{
		Depositor:     agent,
		PartnershipId: "p-dep",
		Amount:        "3000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ = k.GetPartnership(ctx, "p-dep")
	if updated.CommonPotBalance != "9000" { // 6000 + 3000
		t.Errorf("expected pot 9000, got %s", updated.CommonPotBalance)
	}
}

func TestMsgDepositToPot_NotParticipant(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	p := types.Partnership{
		Id:               "p-dep2",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	_, err := srv.DepositToPot(ctx, &types.MsgDepositToPot{
		Depositor:     testAddr(99),
		PartnershipId: "p-dep2",
		Amount:        "1000",
	})
	if err == nil {
		t.Error("expected not participant error")
	}
}

func TestMsgDepositToPot_NotActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	p := types.Partnership{
		Id:               "p-dep3",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusForming,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	_, err := srv.DepositToPot(ctx, &types.MsgDepositToPot{
		Depositor:     testAddr(1),
		PartnershipId: "p-dep3",
		Amount:        "1000",
	})
	if err == nil {
		t.Error("expected not active error")
	}
}

func TestMsgDepositToPotValidateBasic(t *testing.T) {
	depositor := testAddr(1)

	cases := []struct {
		name    string
		msg     types.MsgDepositToPot
		wantErr bool
	}{
		{"valid", types.MsgDepositToPot{Depositor: depositor, PartnershipId: "p-1", Amount: "1000"}, false},
		{"empty depositor", types.MsgDepositToPot{Depositor: "", PartnershipId: "p-1", Amount: "1000"}, true},
		{"empty partnership", types.MsgDepositToPot{Depositor: depositor, PartnershipId: "", Amount: "1000"}, true},
		{"zero amount", types.MsgDepositToPot{Depositor: depositor, PartnershipId: "p-1", Amount: "0"}, true},
		{"negative amount", types.MsgDepositToPot{Depositor: depositor, PartnershipId: "p-1", Amount: "-100"}, true},
		{"invalid amount", types.MsgDepositToPot{Depositor: depositor, PartnershipId: "p-1", Amount: "abc"}, true},
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

// -----------------------------------------------------------------------
// Tests: Withdrawal Execution via ApproveOperation
// -----------------------------------------------------------------------

func TestWithdrawalExecution(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-wd",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "20000",
	}
	k.SetPartnership(ctx, p)

	// Create a withdrawal operation proposed by human, already in approved window
	currentBlock := uint64(ctx.BlockHeight()) // 100
	op := types.ConsensusOperation{
		Id:            "op-wd-1",
		PartnershipId: "p-wd",
		Type:          "withdrawal",
		ProposedBy:    human,
		Amount:        "3000",
		Status:        types.OpStatusProposed,
		Deliberation: &types.DeliberationState{
			AmountTier:   types.AmountTierMedium,
			FloorEndsAt:  currentBlock - 10, // floor already passed
			WindowEndsAt: currentBlock + 100, // window still open
		},
		CreatedAt: currentBlock - 50,
	}
	k.SetConsensusOperation(ctx, op)

	// Agent approves
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.ApproveOperation(ctx, &types.MsgApproveOperation{
		Approver:    agent,
		OperationId: "op-wd-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify pot balance decreased
	updated, _ := k.GetPartnership(ctx, "p-wd")
	if updated.CommonPotBalance != "7000" { // 10000 - 3000
		t.Errorf("expected pot 7000, got %s", updated.CommonPotBalance)
	}

	// Verify operation is approved
	approvedOp, _ := k.GetConsensusOperation(ctx, "op-wd-1")
	if approvedOp.Status != types.OpStatusApproved {
		t.Errorf("expected approved status, got %s", approvedOp.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: Reward Multiplier per Lock Tier
// -----------------------------------------------------------------------

func TestGetLockMultiplier_AllTiers(t *testing.T) {
	cases := []struct {
		tier     uint32
		expected uint64
	}{
		{0, 1000000},  // 1.00x
		{1, 1110000},  // 1.11x
		{2, 1220000},  // 1.22x
		{3, 1440000},  // 1.44x
		{4, 1550000},  // 1.55x
		{5, 1770000},  // 1.77x
		{6, 1770000},  // capped at 5
		{99, 1770000}, // capped at 5
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("tier_%d", tc.tier), func(t *testing.T) {
			m := keeper.GetLockMultiplier(tc.tier)
			if m.Uint64() != tc.expected {
				t.Errorf("expected multiplier %d, got %d", tc.expected, m.Uint64())
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests: SettleCoolingPartnerships
// -----------------------------------------------------------------------

func TestSettleCoolingPartnerships(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	// Cooling partnership with cooldown ended
	p := types.Partnership{
		Id:               "p-cool",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusCooling,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "10000",
		ExitState: &types.ExitState{
			InitiatedBy: testAddr(1),
			InitiatedAt: 50,
			CooldownEnd: 90, // before current block (100)
		},
	}
	k.SetPartnership(ctx, p)

	k.SettleCoolingPartnerships(ctx)

	updated, _ := k.GetPartnership(ctx, "p-cool")
	if updated.Status != types.StatusDissolved {
		t.Errorf("expected dissolved, got %s", updated.Status)
	}
}

func TestSettleCoolingPartnerships_NotYetExpired(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	p := types.Partnership{
		Id:               "p-cool2",
		HumanAddr:        testAddr(1),
		AgentAddr:        testAddr(2),
		Status:           types.StatusCooling,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "10000",
		ExitState: &types.ExitState{
			InitiatedBy: testAddr(1),
			InitiatedAt: 50,
			CooldownEnd: 200, // after current block (100)
		},
	}
	k.SetPartnership(ctx, p)

	k.SettleCoolingPartnerships(ctx)

	updated, _ := k.GetPartnership(ctx, "p-cool2")
	if updated.Status != types.StatusCooling {
		t.Errorf("expected still cooling, got %s", updated.Status)
	}
}

func TestConsensusOperationCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, found := k.GetConsensusOperation(ctx, "op-1")
	if found {
		t.Error("expected not found")
	}

	op := types.ConsensusOperation{
		Id:            "op-1",
		PartnershipId: "p-1",
		Type:          "withdrawal",
		ProposedBy:    testAddr(1),
		Amount:        "5000",
		Status:        types.OpStatusProposed,
		Deliberation: &types.DeliberationState{
			AmountTier:   types.AmountTierSmall,
			WindowEndsAt: 200,
			FloorEndsAt:  150,
		},
		CreatedAt: 100,
	}
	k.SetConsensusOperation(ctx, op)

	got, found := k.GetConsensusOperation(ctx, "op-1")
	if !found {
		t.Fatal("expected to find operation")
	}
	if got.Amount != "5000" {
		t.Errorf("expected 5000, got %s", got.Amount)
	}

	all := k.GetAllConsensusOperations(ctx)
	if len(all) != 1 {
		t.Errorf("expected 1, got %d", len(all))
	}

	k.DeleteConsensusOperation(ctx, "op-1")
	_, found = k.GetConsensusOperation(ctx, "op-1")
	if found {
		t.Error("expected deleted")
	}
}

