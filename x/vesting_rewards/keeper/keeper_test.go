package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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

	k := keeper.NewKeeper(cdc, storeKey, nil, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	// Init genesis with default config
	gs := types.DefaultGenesis()
	k.InitGenesis(ctx, *gs)

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
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   recipient,
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
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
	if schedule.ClaimId != "claim-1" {
		t.Errorf("expected claim-1, got %s", schedule.ClaimId)
	}
	if schedule.Recipient != recipient {
		t.Errorf("expected recipient %s, got %s", recipient, schedule.Recipient)
	}
	if schedule.Status != types.VestingStatusActive {
		t.Errorf("expected active status, got %s", schedule.Status)
	}
	if schedule.TotalAmount != "1000000000000000000" {
		t.Errorf("expected total amount 1000000000000000000, got %s", schedule.TotalAmount)
	}
}

func TestCreateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "not-authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestCreateVesting_InvalidCategory(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    "nonexistent_category",
		Source:      types.SourceVerification,
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

// ---------- PauseVesting Tests ----------

func TestPauseVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	// Create a schedule first
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
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
	if schedule.Status != types.VestingStatusPaused {
		t.Errorf("expected paused status, got %s", schedule.Status)
	}
	if schedule.PausedAtBlock != 1000 {
		t.Errorf("expected paused at block 1000, got %d", schedule.PausedAtBlock)
	}
}

func TestPauseVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
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
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
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
	if schedule.Status != types.VestingStatusActive {
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
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	// Resume without pausing first — keeper silently ignores
	_, err := ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("resume vesting should not error for non-paused: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != types.VestingStatusActive {
		t.Errorf("expected active status unchanged, got %s", schedule.Status)
	}
}

// ---------- AccelerateVesting Tests ----------

func TestAccelerateVesting_Defense(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	accelResp, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:         "authority",
		VestingId:         resp.VestingId,
		AccelerationType:  "defense",
		EvidenceReference: "challenge-1",
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}
	if accelResp.DefenseCount != 1 {
		t.Errorf("expected defense count 1, got %d", accelResp.DefenseCount)
	}
	if accelResp.ReplicationCount != 0 {
		t.Errorf("expected replication count 0, got %d", accelResp.ReplicationCount)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.DefenseCount != 1 {
		t.Errorf("expected defense count 1 in schedule, got %d", schedule.DefenseCount)
	}
}

func TestAccelerateVesting_Replication(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	accelResp, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:         "authority",
		VestingId:         resp.VestingId,
		AccelerationType:  "replication",
		EvidenceReference: "replication-1",
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}
	if accelResp.ReplicationCount != 1 {
		t.Errorf("expected replication count 1, got %d", accelResp.ReplicationCount)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.ReplicationCount != 1 {
		t.Errorf("expected replication count 1, got %d", schedule.ReplicationCount)
	}
}

func TestAccelerateVesting_InvalidType(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	msg := &types.MsgAccelerateVesting{
		Authority:        "authority",
		VestingId:        resp.VestingId,
		AccelerationType: "invalid",
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("expected ValidateBasic to reject invalid acceleration type")
	}
}

func TestAccelerateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:        "not-authority",
		VestingId:        resp.VestingId,
		AccelerationType: "defense",
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
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   recipient,
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	completeResp, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender:    recipient,
		VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting failed: %v", err)
	}
	_ = completeResp

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != types.VestingStatusCompleted {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_ByAuthority(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender:    "authority",
		VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting by authority failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != types.VestingStatusCompleted {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   sdk.AccAddress("recipient1__________").String(),
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender:    sdk.AccAddress("random______________").String(),
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
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   recipient,
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	// Complete once
	ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender: recipient, VestingId: resp.VestingId,
	})

	// Complete again
	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for already completed")
	}
}

func TestCompleteVesting_FalsifiedSchedule(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:   "authority",
		ClaimId:     "claim-1",
		FactId:      "fact-1",
		Recipient:   recipient,
		TotalAmount: "1000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})

	// Manually set to falsified
	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	schedule.Status = types.VestingStatusFalsified
	k.SetVestingSchedule(ctx, schedule)

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for falsified schedule")
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgCreateVesting_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgCreateVesting
		wantErr bool
	}{
		{"valid", types.MsgCreateVesting{
			Authority: "authority", ClaimId: "claim-1",
			Recipient: sdk.AccAddress("recipient1__________").String(),
			TotalAmount: "1000", Category: types.CategoryFormalProof,
		}, false},
		{"missing authority", types.MsgCreateVesting{
			ClaimId: "claim-1",
			Recipient: sdk.AccAddress("recipient1__________").String(),
			TotalAmount: "1000",
		}, true},
		{"missing claim id", types.MsgCreateVesting{
			Authority: "authority",
			Recipient: sdk.AccAddress("recipient1__________").String(),
			TotalAmount: "1000",
		}, true},
		{"zero amount", types.MsgCreateVesting{
			Authority: "authority", ClaimId: "claim-1",
			Recipient: sdk.AccAddress("recipient1__________").String(),
			TotalAmount: "0",
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
		msg     types.MsgAccelerateVesting
		wantErr bool
	}{
		{"valid defense", types.MsgAccelerateVesting{
			Authority: "authority", VestingId: "v1", AccelerationType: "defense",
		}, false},
		{"valid replication", types.MsgAccelerateVesting{
			Authority: "authority", VestingId: "v1", AccelerationType: "replication",
		}, false},
		{"invalid type", types.MsgAccelerateVesting{
			Authority: "authority", VestingId: "v1", AccelerationType: "bad",
		}, true},
		{"missing vesting id", types.MsgAccelerateVesting{
			Authority: "authority", AccelerationType: "defense",
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

// ---------- Full Lifecycle Test ----------

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
func setupKeeperWithBank(t *testing.T, bk *mockBankKeeper, sk *mockStakingKeeper) (keeper.Keeper, sdk.Context) {
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

	k := keeper.NewKeeper(cdc, storeKey, bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	gs := types.DefaultGenesis()
	k.InitGenesis(ctx, *gs)

	return k, ctx
}

// ---------- Block Reward Distribution with Bank Tests ----------

func TestDistributeBlockReward_MintAndDistribute(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22} // Full reward (target = 22)
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Default base reward: 11 LGM = 11000000000000000000 ulgm
	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero total minted")
	}
	if dist.ProducerReward == "0" {
		t.Fatal("expected non-zero producer reward")
	}
	if dist.ResearchShare == "0" {
		t.Fatal("expected non-zero research share")
	}

	// Verify bank keeper was called
	if bk.mintedCoins.IsZero() {
		t.Fatal("expected MintCoins to be called")
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

	// Scaled reward: 11 LGM * 3/22 ≈ 1.5 LGM
	// Total minted should be ~1500000000000000000
	if dist.ValidatorCount != 3 {
		t.Errorf("expected validator count 3, got %d", dist.ValidatorCount)
	}
	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero total minted even with reduced validators")
	}

	// With 3/22 validators, reward should be significantly less than full reward
	// Full: 11000000000000000000, Scaled: 11000000000000000000 * 3/22 = 1500000000000000000
	if dist.TotalMinted == "11000000000000000000" {
		t.Error("expected scaled reward, got full reward amount")
	}
}

func TestDistributeBlockReward_EmptyBlock(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, false)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// PoT alignment: 0% for empty blocks (EmptyBlockRewardRate default = 0)
	if dist.TotalMinted != "0" {
		t.Errorf("expected 0 minted for empty block, got %s", dist.TotalMinted)
	}
	if !bk.mintedCoins.IsZero() {
		t.Error("expected no MintCoins call for empty block")
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

	// Advance past cliff (default cliff = 7777 blocks for formal_proof)
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

	// Default: 22% research share (2200 bps)
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
	if updated.Status != types.VestingStatusFalsified {
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
		Authority:   "authority",
		ClaimId:     "claim-lifecycle",
		FactId:      "fact-lifecycle",
		Recipient:   recipient,
		TotalAmount: "10000000000000000000",
		Category:    types.CategoryFormalProof,
		Source:      types.SourceVerification,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 2. Accelerate (defense)
	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationType: "defense", EvidenceReference: "challenge-1",
	})

	// 3. Pause
	ctx = ctx.WithBlockHeight(2000)
	ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority", VestingId: createResp.VestingId, Reason: "dispute",
	})

	schedule, _ := k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != types.VestingStatusPaused {
		t.Fatalf("expected paused, got %s", schedule.Status)
	}

	// 4. Resume
	ctx = ctx.WithBlockHeight(3000)
	ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: createResp.VestingId,
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != types.VestingStatusActive {
		t.Fatalf("expected active after resume, got %s", schedule.Status)
	}
	if schedule.TotalPausedBlocks != 1000 {
		t.Errorf("expected 1000 paused blocks, got %d", schedule.TotalPausedBlocks)
	}

	// 5. Accelerate (replication)
	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationType: "replication", EvidenceReference: "replication-1",
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.DefenseCount != 1 || schedule.ReplicationCount != 1 {
		t.Errorf("expected 1/1 defense/replication, got %d/%d", schedule.DefenseCount, schedule.ReplicationCount)
	}

	// 6. Complete
	_, err = ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Sender: recipient, VestingId: createResp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != types.VestingStatusCompleted {
		t.Errorf("expected completed, got %s", schedule.Status)
	}
}
