package keeper_test

import (
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/vesting_rewards/keeper"
	"github.com/legible-money/legible/x/vesting_rewards/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.Seal()
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

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), nil, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	// Init genesis with default config
	gs := types.DefaultGenesis()
	k.InitGenesis(ctx, gs)

	return k, ctx
}

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := setupKeeper(t)
	return keeper.NewMsgServerImpl(k), k, ctx
}

// ---------- CreateVesting Tests ----------

func TestCreateVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}
	if resp.VestingId == "" {
		t.Fatal("expected non-empty vesting ID")
	}

	// Verify schedule stored
	schedule, found := k.GetVestingSchedule(ctx, resp.VestingId)
	if !found {
		t.Fatal("schedule not found")
	}
	if schedule.Recipient != recipient {
		t.Errorf("expected recipient %s, got %s", recipient, schedule.Recipient)
	}
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status, got %s", schedule.Status)
	}
	if schedule.TotalAmount != "1000000000000000000" {
		t.Errorf("expected total amount 1000000000000000000, got %s", schedule.TotalAmount)
	}
}

func TestCreateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "not-authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestCreateVesting_InvalidCategory(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	// Unspecified category maps to peer_reviewed by default, which is a valid config
	// so this test now checks that the schedule is created with a valid fallback
	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_UNSPECIFIED,
		LinkedFactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("expected unspecified category to fall back to default, got error: %v", err)
	}
	if resp.VestingId == "" {
		t.Fatal("expected non-empty vesting ID")
	}
}

// ---------- PauseVesting Tests ----------

func TestPauseVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	// Create a schedule first
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	// Pause it
	_, err := ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority",
		VestingId: resp.VestingId,
		Reason:    "active challenge",
	})
	if err != nil {
		t.Fatalf("pause vesting failed: %v", err)
	}

	// Verify paused
	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusPaused) {
		t.Errorf("expected paused status, got %s", schedule.Status)
	}
	if schedule.PausedAtBlock != 1000 {
		t.Errorf("expected paused at block 1000, got %d", schedule.PausedAtBlock)
	}
}

func TestPauseVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "not-authority",
		VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- ResumeVesting Tests ----------

func TestResumeVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	// Pause at block 1000
	ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority", VestingId: resp.VestingId, Reason: "challenge",
	})

	// Resume at block 1100 (100 blocks paused)
	ctx = ctx.WithBlockHeight(1100)
	_, err := ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("resume vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status, got %s", schedule.Status)
	}
	if schedule.TotalPausedBlocks != 100 {
		t.Errorf("expected 100 paused blocks, got %d", schedule.TotalPausedBlocks)
	}
	if schedule.PausedAtBlock != 0 {
		t.Errorf("expected paused_at_block reset to 0, got %d", schedule.PausedAtBlock)
	}
}

func TestResumeVesting_NotPaused(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	// Resume without pausing first -- keeper silently ignores
	_, err := ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("resume vesting should not error for non-paused: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status unchanged, got %s", schedule.Status)
	}
}

// ---------- AccelerateVesting Tests ----------

func TestAccelerateVesting_Defense(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 1000000, // maps to "defense"
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.DefenseCount != 1 {
		t.Errorf("expected defense count 1 in schedule, got %d", schedule.DefenseCount)
	}
}

func TestAccelerateVesting_Replication(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 600000, // maps to "replication"
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.ReplicationCount != 1 {
		t.Errorf("expected replication count 1, got %d", schedule.ReplicationCount)
	}
}

func TestAccelerateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "not-authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 1000000,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- CompleteVesting Tests ----------

func TestCompleteVesting_ByRecipient(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	completeResp, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient,
		VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting failed: %v", err)
	}
	_ = completeResp

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_ByAuthority(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: "authority",
		VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting by authority failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: sdk.AccAddress("random______________").String(),
		VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestCompleteVesting_AlreadyCompleted(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	// Complete once
	ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})

	// Complete again
	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for already completed")
	}
}

func TestCompleteVesting_FalsifiedSchedule(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	// Manually set to falsified
	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	schedule.Status = string(types.VestingStatusFalsified)
	k.SetVestingSchedule(ctx, schedule) // schedule is already a pointer

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for falsified schedule")
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgCreateVesting_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     *types.MsgCreateVesting
		wantErr bool
	}{
		{"valid", &types.MsgCreateVesting{
			Authority:   "authority",
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "1000",
		}, false},
		{"missing authority", &types.MsgCreateVesting{
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "1000",
		}, true},
		{"missing beneficiary", &types.MsgCreateVesting{
			Authority: "authority",
			Amount:    "1000",
		}, true},
		{"zero amount", &types.MsgCreateVesting{
			Authority:   "authority",
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "0",
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

func TestMsgAccelerateVesting_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     *types.MsgAccelerateVesting
		wantErr bool
	}{
		{"valid", &types.MsgAccelerateVesting{
			Authority: "authority", VestingId: "v1", AccelerationFactor: 1000000,
		}, false},
		{"missing vesting id", &types.MsgAccelerateVesting{
			Authority: "authority", AccelerationFactor: 1000000,
		}, true},
		{"missing authority", &types.MsgAccelerateVesting{
			VestingId: "v1", AccelerationFactor: 1000000,
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

// ---------- Mock Bank/Staking Keepers ----------

type mockBankKeeper struct {
	mintedCoins   sdk.Coins
	sentToAccount map[string]sdk.Coins // recipientAddr -> coins
	sentToModule  map[string]sdk.Coins // moduleName -> coins
	mintErr       error
	sendAccErr    error
	sendModErr    error
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		sentToAccount: make(map[string]sdk.Coins),
		sentToModule:  make(map[string]sdk.Coins),
	}
}

func (m *mockBankKeeper) MintCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	if m.mintErr != nil {
		return m.mintErr
	}
	m.mintedCoins = m.mintedCoins.Add(amounts...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if m.sendAccErr != nil {
		return m.sendAccErr
	}
	key := recipientAddr.String()
	m.sentToAccount[key] = m.sentToAccount[key].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _ string, recipientModule string, amt sdk.Coins) error {
	if m.sendModErr != nil {
		return m.sendModErr
	}
	m.sentToModule[recipientModule] = m.sentToModule[recipientModule].Add(amt...)
	return nil
}

type mockStakingKeeper struct {
	activeCount uint32
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ sdk.Context) uint32 {
	return m.activeCount
}

// setupKeeperWithBank sets up a keeper with mock bank and staking keepers.
// Uses default genesis which includes finite fund mode.
func setupKeeperWithBank(t *testing.T, bk *mockBankKeeper, sk *mockStakingKeeper) (keeper.Keeper, sdk.Context) {
	t.Helper()
	return setupKeeperWithBankAndGenesis(t, bk, sk, types.DefaultGenesis())
}

// setupKeeperWithBankAndGenesis sets up a keeper with mock keepers and custom genesis.
func setupKeeperWithBankAndGenesis(t *testing.T, bk *mockBankKeeper, sk *mockStakingKeeper, gs *types.GenesisState) (keeper.Keeper, sdk.Context) {
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

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	k.InitGenesis(ctx, gs)

	return k, ctx
}

// ---------- Block Reward Distribution with Bank Tests ----------

func TestDistributeBlockReward_MintAndDistribute(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22} // Full reward (target = 22)
	k, ctx := setupKeeperWithBank(t, bk, sk)

	// Record fund balance before distribution
	fundBefore := k.GetFundBalance(ctx)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero total minted")
	}
	if dist.ProducerReward == "0" {
		t.Fatal("expected non-zero producer reward")
	}
	if dist.ResearchShare == "0" {
		t.Fatal("expected non-zero research share")
	}

	// Verify fund balance decreased (finite fund deduction, not per-block mint)
	fundAfter := k.GetFundBalance(ctx)
	if fundAfter.Cmp(fundBefore) >= 0 {
		t.Fatal("expected fund balance to decrease after block reward distribution")
	}
	if len(bk.sentToAccount) == 0 {
		t.Fatal("expected SendCoinsFromModuleToAccount to be called for producer")
	}
	if _, ok := bk.sentToModule["research_fund"]; !ok {
		t.Fatal("expected SendCoinsFromModuleToModule to be called for research_fund")
	}
}

func TestDistributeBlockReward_ValidatorScaling(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 3} // 3 of 22 target validators
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 3, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Scaled reward: 11000000 * 3/22 = 1500000
	if dist.ValidatorCount != 3 {
		t.Errorf("expected validator count 3, got %d", dist.ValidatorCount)
	}
	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero total minted even with reduced validators")
	}

	// With 3/22 validators, reward should be less than full block reward (11000000)
	if dist.TotalMinted == "11000000" {
		t.Error("expected scaled reward, got full reward amount")
	}
}

func TestDistributeBlockReward_EmptyBlock(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	// Record fund balance before
	fundBefore := k.GetFundBalance(ctx)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, false)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// PoT alignment: 0% for empty blocks (EmptyBlockRewardRate default = 0)
	if dist.TotalMinted != "0" {
		t.Errorf("expected 0 minted for empty block, got %s", dist.TotalMinted)
	}
	// Fund balance should be unchanged for empty block
	fundAfter := k.GetFundBalance(ctx)
	if fundAfter.Cmp(fundBefore) != 0 {
		t.Errorf("expected fund balance unchanged for empty block, before=%s after=%s", fundBefore.String(), fundAfter.String())
	}
}

func TestDistributeBlockReward_NilBankKeeper(t *testing.T) {
	// Keeper with nil bank (original behavior)
	k, ctx := setupKeeper(t)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Should still calculate and record distribution even without bank
	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero distribution record even without bank")
	}
}

// ---------- Claim Rewards with Bank Tests ----------

func TestClaimRewards_SendsCoins(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Create a vesting schedule
	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	// Advance past cliff (default cliff = 5555 blocks for formal_proof)
	ctx = ctx.WithBlockHeight(20000)

	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err != nil {
		t.Fatalf("claim rewards failed: %v", err)
	}

	if claimed == "0" {
		t.Fatal("expected non-zero claimed amount after cliff")
	}

	// Verify bank keeper received SendCoinsFromModuleToAccount
	if _, ok := bk.sentToAccount[recipient]; !ok {
		t.Fatal("expected SendCoinsFromModuleToAccount to be called for recipient")
	}
}

func TestClaimRewards_NilBankKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	// Advance past cliff
	ctx = ctx.WithBlockHeight(20000)

	// Should succeed without bank keeper (just updates state)
	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err != nil {
		t.Fatalf("claim rewards with nil bank failed: %v", err)
	}
	if claimed == "0" {
		t.Fatal("expected non-zero claimed amount")
	}
}

// ---------- Route Reward Tests ----------

func TestRouteReward_ResearchSplit(t *testing.T) {
	k, ctx := setupKeeper(t)

	routing, err := k.RouteReward(ctx, types.SourceBlockProduction, "10000", sdk.AccAddress("recipient___________").String(), "")
	if err != nil {
		t.Fatalf("route reward failed: %v", err)
	}

	// Default: 22% research share (220000 bps on 1000000 scale)
	if routing.ResearchShare != "2200" {
		t.Errorf("expected research share 2200, got %s", routing.ResearchShare)
	}
	if routing.RecipientAmount != "7800" {
		t.Errorf("expected recipient amount 7800, got %s", routing.RecipientAmount)
	}
}

func TestRouteReward_InvalidAmount(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.RouteReward(ctx, types.SourceBlockProduction, "0", "addr", "")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}

	_, err = k.RouteReward(ctx, types.SourceBlockProduction, "notanumber", "addr", "")
	if err == nil {
		t.Fatal("expected error for non-numeric amount")
	}
}

// ---------- Falsify Claim Tests ----------

func TestFalsifyClaim_ClawbackCalculation(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	recipient := sdk.AccAddress("recipient1__________").String()

	schedule, err := k.CreateVestingSchedule(ctx, "claim-falsify", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	// Advance and claim some rewards
	ctx = ctx.WithBlockHeight(20000)
	k.ClaimRewards(ctx, recipient, schedule.Id)

	// Falsify the claim
	challenger := sdk.AccAddress("challenger__________").String()
	record, err := k.FalsifyClaim(ctx, "claim-falsify", challenger)
	if err != nil {
		t.Fatalf("falsify claim failed: %v", err)
	}

	if record.VestingId != schedule.Id {
		t.Errorf("expected vesting ID %s, got %s", schedule.Id, record.VestingId)
	}

	// Verify schedule is now falsified
	updated, _ := k.GetVestingSchedule(ctx, schedule.Id)
	if updated.Status != string(types.VestingStatusFalsified) {
		t.Errorf("expected falsified status, got %s", updated.Status)
	}
}

func TestFalsifyClaim_AlreadyFalsified(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	k.CreateVestingSchedule(ctx, "claim-dup", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)

	// Falsify once
	_, err := k.FalsifyClaim(ctx, "claim-dup", "challenger")
	if err != nil {
		t.Fatalf("first falsify failed: %v", err)
	}

	// Falsify again should error
	_, err = k.FalsifyClaim(ctx, "claim-dup", "challenger")
	if err == nil {
		t.Fatal("expected error for already falsified claim")
	}
}

// ---------- Full Lifecycle Test ----------

func TestVestingFullLifecycle(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// 1. Create
	createResp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "10000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-lifecycle",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 2. Accelerate (defense via high factor)
	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationFactor: 1000000, // maps to "defense"
	})

	// 3. Pause
	ctx = ctx.WithBlockHeight(2000)
	ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority", VestingId: createResp.VestingId, Reason: "dispute",
	})

	schedule, _ := k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusPaused) {
		t.Fatalf("expected paused, got %s", schedule.Status)
	}

	// 4. Resume
	ctx = ctx.WithBlockHeight(3000)
	ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: createResp.VestingId,
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Fatalf("expected active after resume, got %s", schedule.Status)
	}
	if schedule.TotalPausedBlocks != 1000 {
		t.Errorf("expected 1000 paused blocks, got %d", schedule.TotalPausedBlocks)
	}

	// 5. Accelerate (replication via mid-range factor)
	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationFactor: 600000, // maps to "replication"
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.DefenseCount != 1 || schedule.ReplicationCount != 1 {
		t.Errorf("expected 1/1 defense/replication, got %d/%d", schedule.DefenseCount, schedule.ReplicationCount)
	}

	// 6. Complete
	_, err = ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: createResp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed, got %s", schedule.Status)
	}
}

// ==================== Finite Fund Tests ====================

// setupFiniteFundKeeper creates a keeper with finite fund mode enabled using custom params.
func setupFiniteFundKeeper(t *testing.T, bk *mockBankKeeper, initialFund string, blockHeight int64) (keeper.Keeper, sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.Params.InitialFundBalance = initialFund
	gs.Params.BlockRewardAmount = "11000000"    // 11 LGM initial reward
	gs.Params.BlocksPerEpoch = 240000
	gs.Params.DecayFactorBps = 900000            // 0.9 per epoch
	gs.Params.FloorReward = "100000"              // 0.1 LGM floor
	gs.Params.FundHealthThresholdBps = 100000     // 10%

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	sk := &mockStakingKeeper{activeCount: 22}

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: blockHeight}, false, log.NewNopLogger())

	k.InitGenesis(ctx, gs)

	return k, ctx
}

func TestFiniteFundReward_Epoch0(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At epoch 0, no decay. Reward = 11000000 (full block reward).
	if dist.TotalMinted != "11000000" {
		t.Errorf("expected full reward 11000000 at epoch 0, got %s", dist.TotalMinted)
	}
}

func TestFiniteFundReward_Epoch1(t *testing.T) {
	bk := newMockBankKeeper()
	// Block 240000 = start of epoch 1
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 240000)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At epoch 1: 11000000 * 0.9 = 9900000
	if dist.TotalMinted != "9900000" {
		t.Errorf("expected decayed reward 9900000 at epoch 1, got %s", dist.TotalMinted)
	}
}

func TestFiniteFundReward_Epoch10(t *testing.T) {
	bk := newMockBankKeeper()
	// Block 2400000 = start of epoch 10
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 2400000)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	totalMinted := dist.TotalMinted
	if totalMinted == "0" || totalMinted == "11000000" {
		t.Errorf("expected decayed reward ~3834863 at epoch 10, got %s", totalMinted)
	}
	// Should be between 3500000 and 4000000
	minted := new(big.Int)
	minted.SetString(totalMinted, 10)
	if minted.Cmp(big.NewInt(3500000)) < 0 || minted.Cmp(big.NewInt(4000000)) > 0 {
		t.Errorf("epoch 10 reward %s outside expected range [3500000, 4000000]", totalMinted)
	}
}

func TestFiniteFundReward_FloorReward(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 200*240000)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At epoch 200, decay brings reward well below floor, so floor kicks in
	if dist.TotalMinted != "100000" {
		t.Errorf("expected floor reward 100000, got %s", dist.TotalMinted)
	}
}

func TestFiniteFundReward_FundExhausted(t *testing.T) {
	bk := newMockBankKeeper()
	// Tiny fund that will be exhausted
	k, ctx := setupFiniteFundKeeper(t, bk, "100", 0)

	producer := sdk.AccAddress("producer____________").String()

	// First call: reward would be 11000000 but fund is only 100 -- capped to 100
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist.TotalMinted != "100" {
		t.Errorf("expected reward capped to fund balance 100, got %s", dist.TotalMinted)
	}

	// Second call: fund should be exhausted (0 left)
	dist2, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist2.TotalMinted != "0" {
		t.Errorf("expected 0 reward when fund exhausted, got %s", dist2.TotalMinted)
	}
}

func TestFiniteFundReward_HealthTaper(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "1000000", 0)

	// Manually set fund balance to 50000 (simulating depletion)
	k.SetFundBalance(ctx, big.NewInt(50000))

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dist.TotalMinted == "0" {
		t.Error("expected non-zero reward when fund has balance")
	}
	minted := new(big.Int)
	minted.SetString(dist.TotalMinted, 10)
	if minted.Cmp(big.NewInt(50000)) > 0 {
		t.Errorf("expected reward capped to fund balance 50000, got %s", dist.TotalMinted)
	}
}

func TestFiniteFundReward_CappedToBalance(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "5000000", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reward should be capped to fund balance
	if dist.TotalMinted != "5000000" {
		t.Errorf("expected reward capped to 5000000, got %s", dist.TotalMinted)
	}

	// Fund should now be 0
	remaining := k.GetFundBalance(ctx)
	if remaining.Sign() != 0 {
		t.Errorf("expected 0 fund balance after full deduction, got %s", remaining.String())
	}
}

func TestApplyDecay(t *testing.T) {
	bk := newMockBankKeeper()

	tests := []struct {
		name      string
		epoch     int64
		minReward int64
		maxReward int64
	}{
		{"epoch 0", 0, 11000000, 11000000},
		{"epoch 1", 240000, 9900000, 9900000},
		{"epoch 2", 480000, 8910000, 8910000},
		{"epoch 5", 1200000, 6000000, 6600000},
		{"epoch 10", 2400000, 3500000, 4000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bk := newMockBankKeeper()
			k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", tt.epoch)

			producer := sdk.AccAddress("producer____________").String()
			dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			minted := new(big.Int)
			minted.SetString(dist.TotalMinted, 10)
			if minted.Cmp(big.NewInt(tt.minReward)) < 0 || minted.Cmp(big.NewInt(tt.maxReward)) > 0 {
				t.Errorf("epoch reward %s outside expected range [%d, %d]", dist.TotalMinted, tt.minReward, tt.maxReward)
			}
		})
	}
	_ = bk // suppress unused
}

func TestFundBalanceGetSet(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetFundBalance(ctx, big.NewInt(999999))
	bal := k.GetFundBalance(ctx)
	if bal.Int64() != 999999 {
		t.Errorf("expected 999999, got %s", bal.String())
	}

	// Deduct
	ok := k.DeductFromFund(ctx, big.NewInt(500000))
	if !ok {
		t.Fatal("expected successful deduction")
	}
	bal = k.GetFundBalance(ctx)
	if bal.Int64() != 499999 {
		t.Errorf("expected 499999 after deduction, got %s", bal.String())
	}

	// Deduct more than balance -- should fail
	ok = k.DeductFromFund(ctx, big.NewInt(999999))
	if ok {
		t.Fatal("expected failed deduction (insufficient balance)")
	}
	// Balance should be unchanged
	bal = k.GetFundBalance(ctx)
	if bal.Int64() != 499999 {
		t.Errorf("expected balance unchanged at 499999 after failed deduction, got %s", bal.String())
	}
}

func TestExportGenesis_PreservesFundBalance(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 0)

	producer := sdk.AccAddress("producer____________").String()
	// Distribute one block reward to reduce fund
	k.DistributeBlockReward(ctx, producer, 22, true)

	// Export genesis
	exported := k.ExportGenesis(ctx)

	// InitialFundBalance should be current balance (less than original)
	if exported.Params.InitialFundBalance == "133333333200000" {
		t.Error("expected exported fund balance to be less than initial after distributing rewards")
	}
	if exported.Params.InitialFundBalance == "0" || exported.Params.InitialFundBalance == "" {
		t.Error("expected non-zero exported fund balance")
	}
}

func TestFiniteFundReward_FundBalanceAfterField(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFiniteFundKeeper(t, bk, "133333333200000", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// FundBalanceAfter should be set in finite fund mode
	if dist.FundBalanceAfter == "" {
		t.Error("expected FundBalanceAfter to be set in finite fund mode")
	}
	// Should be initial - totalMinted
	expected := new(big.Int)
	expected.SetString("133333333200000", 10)
	minted := new(big.Int)
	minted.SetString(dist.TotalMinted, 10)
	expected.Sub(expected, minted)

	if dist.FundBalanceAfter != expected.String() {
		t.Errorf("expected FundBalanceAfter %s, got %s", expected.String(), dist.FundBalanceAfter)
	}
}
