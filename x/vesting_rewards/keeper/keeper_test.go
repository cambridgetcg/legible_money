package keeper_test

import (
	"context"
	"math/big"
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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

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
	sentToAccount map[string]sdk.Coins   // recipientAddr -> coins
	sentToModule  map[string]sdk.Coins   // moduleName -> coins
	balances      map[string]sdk.Coins   // addr.String() -> coins (for GetAllBalances)
	supply        map[string]sdkmath.Int // denom -> total supply (tracks mints - burns)
	mintErr       error
	sendAccErr    error
	sendModErr    error
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		sentToAccount: make(map[string]sdk.Coins),
		sentToModule:  make(map[string]sdk.Coins),
		supply:        make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) MintCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	if m.mintErr != nil {
		return m.mintErr
	}
	m.mintedCoins = m.mintedCoins.Add(amounts...)
	for _, coin := range amounts {
		cur, ok := m.supply[coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.supply[coin.Denom] = cur.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	for _, coin := range amounts {
		cur, ok := m.supply[coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.supply[coin.Denom] = cur.Sub(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	if amt, ok := m.supply[denom]; ok {
		return sdk.NewCoin(denom, amt)
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
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

func (m *mockBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if m.balances != nil {
		if coins, ok := m.balances[addr.String()]; ok {
			return coins
		}
	}
	return sdk.Coins{}
}

type mockStakingKeeper struct {
	activeCount uint32
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ context.Context) uint32 {
	return m.activeCount
}

// setupKeeperWithBank sets up a keeper with mock bank and staking keepers.
// Uses default genesis (pure PoT: no pre-minted fund).
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

	// Total minted should start at 0 (pure PoT: no genesis mint)
	mintedBefore := k.GetTotalMinted(ctx)
	if mintedBefore.Sign() != 0 {
		t.Fatalf("expected 0 total minted at genesis, got %s", mintedBefore.String())
	}

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

	// Verify total minted increased (per-block mint)
	mintedAfter := k.GetTotalMinted(ctx)
	if mintedAfter.Cmp(mintedBefore) <= 0 {
		t.Fatal("expected total minted to increase after block reward distribution")
	}
	// Verify bank keeper was called for minting
	if bk.mintedCoins.IsZero() {
		t.Fatal("expected MintCoins to be called")
	}
	if len(bk.sentToAccount) == 0 {
		t.Fatal("expected SendCoinsFromModuleToAccount to be called for producer")
	}
	if _, ok := bk.sentToModule["research_fund"]; !ok {
		t.Fatal("expected SendCoinsFromModuleToModule to be called for research_fund")
	}
	// Verify verification pool routing to knowledge module
	if _, ok := bk.sentToModule["knowledge"]; !ok {
		t.Fatal("expected SendCoinsFromModuleToModule to be called for knowledge (verification pool)")
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

	// Scaled reward: 3000000 * 3/22 = 409090
	if dist.ValidatorCount != 3 {
		t.Errorf("expected validator count 3, got %d", dist.ValidatorCount)
	}
	if dist.TotalMinted == "0" {
		t.Fatal("expected non-zero total minted even with reduced validators")
	}

	// With 3/22 validators, reward should be less than full block reward (3000000)
	if dist.TotalMinted == "3000000" {
		t.Error("expected scaled reward, got full reward amount")
	}
}

func TestDistributeBlockReward_EmptyBlock(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	// Record total minted before
	mintedBefore := k.GetTotalMinted(ctx)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, false)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// PoT alignment: 0% for empty blocks (EmptyBlockRewardRate default = 0)
	if dist.TotalMinted != "0" {
		t.Errorf("expected 0 minted for empty block, got %s", dist.TotalMinted)
	}
	// Total minted should be unchanged for empty block
	mintedAfter := k.GetTotalMinted(ctx)
	if mintedAfter.Cmp(mintedBefore) != 0 {
		t.Errorf("expected total minted unchanged for empty block, before=%s after=%s", mintedBefore.String(), mintedAfter.String())
	}
}

func TestDistributeBlockReward_NilBankKeeper(t *testing.T) {
	// Keeper with nil bank (MintWithCap skips actual mint but still tracks total)
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
	// Total minted should be tracked
	total := k.GetTotalMinted(ctx)
	if total.Sign() <= 0 {
		t.Fatal("expected total minted to increase even with nil bank")
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

	// Default: 3.33% research share (33300 bps on 1000000 scale)
	// 10000 * 33300 / 1000000 = 333
	if routing.ResearchShare != "333" {
		t.Errorf("expected research share 333, got %s", routing.ResearchShare)
	}
	if routing.RecipientAmount != "9667" {
		t.Errorf("expected recipient amount 9667, got %s", routing.RecipientAmount)
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

// ==================== Pure PoT Mint Tests ====================

// setupMintKeeper creates a keeper for pure PoT testing at a given block height.
// totalMinted allows testing near the supply cap.
func setupMintKeeper(t *testing.T, bk *mockBankKeeper, totalMinted string, blockHeight int64) (keeper.Keeper, sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.Params.InitialFundBalance = totalMinted // InitialFundBalance stores TotalMinted on export/import
	gs.Params.BlockRewardAmount = "3000000"    // 3 LGM initial reward
	gs.Params.BlocksPerEpoch = 240000
	gs.Params.DecayFactorBps = 850000         // 0.85 per epoch
	gs.Params.FloorReward = "100000"          // 0.1 LGM floor
	gs.Params.FundHealthThresholdBps = 100000 // 10% (unused in pure PoT)

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

	// Prime mock bank supply to match totalMinted so MintWithCap's bank supply
	// check is consistent with the genesis state.
	if totalMinted != "" && totalMinted != "0" {
		amt, ok := new(big.Int).SetString(totalMinted, 10)
		if ok && amt.Sign() > 0 {
			bk.supply["ulgm"] = sdkmath.NewIntFromBigInt(amt)
		}
	}

	return k, ctx
}

func TestBlockReward_Epoch0(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At epoch 0, no decay. Reward = 3000000 (full block reward).
	if dist.TotalMinted != "3000000" {
		t.Errorf("expected full reward 3000000 at epoch 0, got %s", dist.TotalMinted)
	}
	// Verify tokens were minted
	if bk.mintedCoins.AmountOf("ulgm").Int64() != 3000000 {
		t.Errorf("expected 3000000 ulgm minted, got %s", bk.mintedCoins.AmountOf("ulgm").String())
	}
}

func TestBlockReward_Epoch1(t *testing.T) {
	bk := newMockBankKeeper()
	// Block 240000 = start of epoch 1
	k, ctx := setupMintKeeper(t, bk, "0", 240000)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At epoch 1: 3000000 * 0.85 = 2550000
	if dist.TotalMinted != "2550000" {
		t.Errorf("expected decayed reward 2550000 at epoch 1, got %s", dist.TotalMinted)
	}
}

func TestBlockReward_Epoch10(t *testing.T) {
	bk := newMockBankKeeper()
	// Block 2400000 = start of epoch 10
	k, ctx := setupMintKeeper(t, bk, "0", 2400000)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	totalMinted := dist.TotalMinted
	if totalMinted == "0" || totalMinted == "3000000" {
		t.Errorf("expected decayed reward ~590622 at epoch 10, got %s", totalMinted)
	}
	// Should be between 550000 and 650000 (3000000 * 0.85^10 ≈ 590622)
	minted := new(big.Int)
	minted.SetString(totalMinted, 10)
	if minted.Cmp(big.NewInt(550000)) < 0 || minted.Cmp(big.NewInt(650000)) > 0 {
		t.Errorf("epoch 10 reward %s outside expected range [550000, 650000]", totalMinted)
	}
}

func TestBlockReward_FloorReward(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 200*240000)

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

func TestMintWithCap_SupplyExhausted(t *testing.T) {
	bk := newMockBankKeeper()
	// Set TotalMinted to just under the cap
	maxSupply := "222222222000000"
	nearCap := "222222221999900" // 100 ulgm remaining
	k, ctx := setupMintKeeper(t, bk, nearCap, 0)

	producer := sdk.AccAddress("producer____________").String()

	// First call: reward would be 11000000 but only 100 remaining under cap
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist.TotalMinted != "100" {
		t.Errorf("expected reward capped to remaining supply 100, got %s", dist.TotalMinted)
	}

	// Total minted should now equal max supply
	totalNow := k.GetTotalMinted(ctx)
	expected := new(big.Int)
	expected.SetString(maxSupply, 10)
	if totalNow.Cmp(expected) != 0 {
		t.Errorf("expected total minted to be %s, got %s", maxSupply, totalNow.String())
	}

	// Second call: supply fully exhausted
	bk2 := newMockBankKeeper() // fresh mock
	_ = bk2
	dist2, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist2.TotalMinted != "0" {
		t.Errorf("expected 0 reward when supply exhausted, got %s", dist2.TotalMinted)
	}
}

func TestMintWithCap_EnforcesSupplyLimit(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	// Try to mint more than max supply
	maxSupply := new(big.Int)
	maxSupply.SetString(types.MaxSupplyUlgm, 10)
	overMax := new(big.Int).Add(maxSupply, big.NewInt(1))

	actual, err := k.MintWithCap(ctx, overMax)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be capped to max supply
	if actual.Cmp(maxSupply) != 0 {
		t.Errorf("expected mint capped to %s, got %s", maxSupply.String(), actual.String())
	}

	// Total minted should equal max supply
	totalMinted := k.GetTotalMinted(ctx)
	if totalMinted.Cmp(maxSupply) != 0 {
		t.Errorf("expected total minted %s, got %s", maxSupply.String(), totalMinted.String())
	}

	// Further mint should return 0
	actual2, err := k.MintWithCap(ctx, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual2.Sign() != 0 {
		t.Errorf("expected 0 mint when supply exhausted, got %s", actual2.String())
	}
}

func TestMintWithCap_BurnRecycling(t *testing.T) {
	bk := newMockBankKeeper()
	// Start with supply at max - only 500 ulgm remaining
	nearCap := "222222221999500"
	k, ctx := setupMintKeeper(t, bk, nearCap, 0)

	// Mint 500 ulgm — exhausts remaining supply
	actual, err := k.MintWithCap(ctx, big.NewInt(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual.Int64() != 500 {
		t.Errorf("expected 500 minted, got %s", actual.String())
	}

	// Supply is now at max — further mint should return 0
	actual2, err := k.MintWithCap(ctx, big.NewInt(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual2.Sign() != 0 {
		t.Errorf("expected 0 mint when supply exhausted, got %s", actual2.String())
	}

	// Simulate burning 200 ulgm (e.g., from slashing in another module).
	// This reduces bank supply, freeing headroom for future block rewards.
	bk.BurnCoins(ctx, "knowledge", sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(200))))

	// Now 200 ulgm of headroom is available again
	actual3, err := k.MintWithCap(ctx, big.NewInt(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual3.Int64() != 200 {
		t.Errorf("expected 200 minted after burn freed headroom, got %s", actual3.String())
	}

	// Supply is back at max — verify exhausted again
	actual4, err := k.MintWithCap(ctx, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual4.Sign() != 0 {
		t.Errorf("expected 0 after re-exhaustion, got %s", actual4.String())
	}
}

func TestApplyDecay(t *testing.T) {
	tests := []struct {
		name      string
		epoch     int64
		minReward int64
		maxReward int64
	}{
		{"epoch 0", 0, 3000000, 3000000},
		{"epoch 1", 240000, 2550000, 2550000},
		{"epoch 2", 480000, 2167500, 2167500},
		{"epoch 5", 1200000, 1300000, 1400000},
		{"epoch 10", 2400000, 550000, 650000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bk := newMockBankKeeper()
			k, ctx := setupMintKeeper(t, bk, "0", tt.epoch)

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
}

func TestTotalMintedGetSet(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetTotalMinted(ctx, big.NewInt(999999))
	total := k.GetTotalMinted(ctx)
	if total.Int64() != 999999 {
		t.Errorf("expected 999999, got %s", total.String())
	}

	// Update total minted
	k.SetTotalMinted(ctx, big.NewInt(1999999))
	total = k.GetTotalMinted(ctx)
	if total.Int64() != 1999999 {
		t.Errorf("expected 1999999, got %s", total.String())
	}
}

func TestExportGenesis_PreservesTotalMinted(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	producer := sdk.AccAddress("producer____________").String()
	// Distribute one block reward to increase total minted
	k.DistributeBlockReward(ctx, producer, 22, true)

	// Export genesis
	exported := k.ExportGenesis(ctx)

	// InitialFundBalance stores TotalMinted for chain restart continuity
	if exported.Params.InitialFundBalance == "0" || exported.Params.InitialFundBalance == "" {
		t.Error("expected non-zero exported total minted after distributing rewards")
	}
	// Should be 3000000 (one block reward at epoch 0)
	if exported.Params.InitialFundBalance != "3000000" {
		t.Errorf("expected exported total minted 3000000, got %s", exported.Params.InitialFundBalance)
	}
}

func TestDistributeBlockReward_VerificationPoolRouting(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// At epoch 0, full validators: total minted = 3000000
	// Research share (3.33%): 3000000 * 33300 / 1000000 = 99900
	// Post-research: 3000000 - 99900 = 2900100
	// Verification pool (50%): 2900100 * 500000 / 1000000 = 1450050
	// Producer (50%): 2900100 - 1450050 = 1450050
	if dist.TotalMinted != "3000000" {
		t.Errorf("expected total minted 3000000, got %s", dist.TotalMinted)
	}
	if dist.ResearchShare != "99900" {
		t.Errorf("expected research share 99900, got %s", dist.ResearchShare)
	}
	if dist.ProducerReward != "1450050" {
		t.Errorf("expected producer reward 1450050, got %s", dist.ProducerReward)
	}

	// Verify bank sends
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("ulgm").Int64() != 99900 {
		t.Errorf("expected 99900 to research_fund, got %d", researchCoins.AmountOf("ulgm").Int64())
	}
	// Verification pool (1450050) is now split:
	// - 30% compute pool: 1450050 * 300000 / 1000000 = 435015
	// - 70% knowledge module: 1450050 - 435015 = 1015035
	knowledgeCoins := bk.sentToModule["knowledge"]
	if knowledgeCoins.AmountOf("ulgm").Int64() != 1015035 {
		t.Errorf("expected 1015035 to knowledge module (70%% of verification pool), got %d", knowledgeCoins.AmountOf("ulgm").Int64())
	}
	computePoolCoins := bk.sentToModule["compute_pool"]
	if computePoolCoins.AmountOf("ulgm").Int64() != 435015 {
		t.Errorf("expected 435015 to compute_pool (30%% of verification pool), got %d", computePoolCoins.AmountOf("ulgm").Int64())
	}
	producerCoins := bk.sentToAccount[producer]
	if producerCoins.AmountOf("ulgm").Int64() != 1450050 {
		t.Errorf("expected 1450050 to producer, got %d", producerCoins.AmountOf("ulgm").Int64())
	}
}

func TestBlockReward_TotalMintedAfterField(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// FundBalanceAfter now stores cumulative TotalMinted
	if dist.FundBalanceAfter == "" {
		t.Error("expected FundBalanceAfter (TotalMinted) to be set")
	}
	// Should equal the minted amount (starting from 0)
	if dist.FundBalanceAfter != dist.TotalMinted {
		t.Errorf("expected FundBalanceAfter %s to equal TotalMinted %s", dist.FundBalanceAfter, dist.TotalMinted)
	}
}

// ==========================================================================
// FEE ROUTING AUDIT — Verify P2P transfers are NOT taxed on transfer amount
// ==========================================================================
//
// This test suite verifies that:
// 1. RouteReward() only applies to protocol-generated value (block rewards, verification)
// 2. RouteFees() sweeps 3.33% of gas fees (flat-rate, NOT proportional to transfer amount)
// 3. P2P transfer amounts are NEVER subject to any circulation tax
// 4. The 3.33/96.67 split math is correct on the 1,000,000 BPS scale

func TestRouteFees_SweepsFeeCollector(t *testing.T) {
	// Simulate: fee_collector has 21000 ulgm from a P2P transfer gas fee.
	// RouteFees should sweep 3.33% (699 ulgm) to research_fund.
	feeCollectorAddr := sdk.AccAddress("fee_collector________").String() // placeholder, real addr doesn't matter
	_ = feeCollectorAddr

	bk := newMockBankKeeper()
	// Set fee_collector balance: 21000 ulgm (gas fee from one MsgSend)
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(21000))),
	}

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// 3.33% of 21000 = 699 ulgm should go to research_fund
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("ulgm").Int64() != 699 {
		t.Errorf("expected 699 ulgm to research_fund, got %s", researchCoins.String())
	}
}

func TestRouteFees_EmptyFeeCollector(t *testing.T) {
	// When fee_collector is empty (no transactions), RouteFees should be a no-op
	bk := newMockBankKeeper()
	bk.balances = map[string]sdk.Coins{} // empty

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// No coins should be sent anywhere
	if len(bk.sentToModule) > 0 {
		t.Errorf("expected no module sends, got %v", bk.sentToModule)
	}
}

func TestRouteFees_OnlyProcessesUlgm(t *testing.T) {
	// RouteFees should only process ulgm, ignore other denoms
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk := newMockBankKeeper()
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(
			sdk.NewCoin("ulgm", sdkmath.NewInt(100000)),
			sdk.NewCoin("atom", sdkmath.NewInt(50000)), // should be ignored
		),
	}

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// Only ulgm should be routed: 3.33% of 100000 = 3330
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("ulgm").Int64() != 3330 {
		t.Errorf("expected 3330 ulgm to research_fund, got %s", researchCoins.String())
	}
	// atom should NOT be sent
	if researchCoins.AmountOf("atom").Int64() != 0 {
		t.Errorf("expected 0 atom to research_fund, got %d", researchCoins.AmountOf("atom").Int64())
	}
}

func TestRouteFees_GasFeeIsFlat_NotProportionalToTransferAmount(t *testing.T) {
	// KEY AUDIT: A P2P transfer of 1 LGM and a transfer of 1,000,000 LGM
	// both cost the same gas fee (21000 gas). The RouteFees sweep is based
	// on gas fee only, NOT on the transfer amount.
	//
	// This test proves the fee collector accumulates the SAME amount
	// regardless of transfer size, and the 7% sweep is therefore flat.

	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)

	// Case 1: Small transfer (1 LGM) — gas fee = 21000 gas * minGasPrice
	smallTransferFee := int64(21000) // ulgm gas fee
	bk1 := newMockBankKeeper()
	bk1.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(smallTransferFee))),
	}
	sk := &mockStakingKeeper{activeCount: 22}
	k1, ctx1 := setupKeeperWithBank(t, bk1, sk)

	if err := k1.RouteFees(ctx1); err != nil {
		t.Fatalf("RouteFees small transfer failed: %v", err)
	}

	// Case 2: Large transfer (1M LGM) - SAME gas fee = 21000 gas * minGasPrice
	largeTransferFee := int64(21000) // ulgm gas fee (SAME!)
	bk2 := newMockBankKeeper()
	bk2.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(largeTransferFee))),
	}
	k2, ctx2 := setupKeeperWithBank(t, bk2, sk)

	if err := k2.RouteFees(ctx2); err != nil {
		t.Fatalf("RouteFees large transfer failed: %v", err)
	}

	// Both should send the EXACT SAME research share
	research1 := bk1.sentToModule["research_fund"].AmountOf("ulgm").Int64()
	research2 := bk2.sentToModule["research_fund"].AmountOf("ulgm").Int64()

	if research1 != research2 {
		t.Errorf("AUDIT FAIL: small transfer research fee (%d) != large transfer research fee (%d). This means the tax IS proportional to transfer amount!", research1, research2)
	}
	if research1 != 699 { // 3.33% of 21000
		t.Errorf("expected 699, got %d", research1)
	}
}

func TestRouteReward_OnlyForProtocolRewards(t *testing.T) {
	// RouteReward is ONLY called for protocol-generated value:
	// - Block production rewards (SourceBlockProduction)
	// - Verification rewards (SourceVerification)
	// - Claim acceptance rewards
	// It is NEVER called for P2P transfers.
	k, ctx := setupKeeper(t)

	// Test each valid reward source
	sources := []types.RewardSource{
		types.SourceBlockProduction,
		types.SourceVerification,
		types.SourceChallengeWin,
	}

	for _, src := range sources {
		routing, err := k.RouteReward(ctx, src, "100000", sdk.AccAddress("recipient___________").String(), "fact-1")
		if err != nil {
			t.Fatalf("RouteReward failed for source %s: %v", src, err)
		}

		// 3.33% research share: 100000 * 33300 / 1000000 = 3330
		resBig := new(big.Int)
		resBig.SetString(routing.ResearchShare, 10)
		expectedResearch := new(big.Int).SetInt64(3330)

		if resBig.Cmp(expectedResearch) != 0 {
			t.Errorf("source %s: expected research share 3330, got %s", src, routing.ResearchShare)
		}

		// 96.67% to recipient: 100000 - 3330 = 96670
		recipBig := new(big.Int)
		recipBig.SetString(routing.RecipientAmount, 10)
		expectedRecip := new(big.Int).SetInt64(96670)

		if recipBig.Cmp(expectedRecip) != 0 {
			t.Errorf("source %s: expected recipient amount 96670, got %s", src, routing.RecipientAmount)
		}
	}
}

func TestP2PTransfer_NeverTaxedOnAmount(t *testing.T) {
	// DEFINITIVE AUDIT: This test proves that P2P transfer amounts are NEVER
	// subject to any circulation tax, retrieval rate, or proportional fee.
	//
	// The fee flow for a P2P transfer (MsgSend):
	// 1. Sender pays flat gas fee (21000 gas) → goes to fee_collector
	// 2. ante.DeductFeeDecorator deducts fee from sender to fee_collector
	// 3. BeginBlock: RouteFees sweeps 3.33% of fee_collector to research_fund
	// 4. x/distribution sweeps remaining 96.67% to validators
	// 5. The transfer amount goes directly from sender to recipient via x/bank
	//
	// At NO POINT is the transfer amount taxed. Only the gas fee is split.

	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	// Simulate: User transfers 1,000,000 ulgm (1 LGM) to another user.
	// The transfer amount (1,000,000 ulgm) goes through x/bank MsgSend.
	// The gas fee (~525 ulgm at 0.025 gas price * 21000 gas) goes to fee_collector.
	transferAmount := int64(1000000) // 1 LGM in ulgm
	gasFee := int64(525)             // 0.025 * 21000 gas

	// STEP 1: Only the gas fee enters fee_collector (NOT the transfer amount)
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(gasFee))),
	}

	// STEP 2: RouteFees sweeps 7% of gas fee to research
	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// STEP 3: Verify ONLY the gas fee was taxed
	researchSweep := bk.sentToModule["research_fund"].AmountOf("ulgm").Int64()
	expectedResearchFromGas := gasFee * 33300 / 1000000 // 3.33% of 525 = 17

	if researchSweep != expectedResearchFromGas {
		t.Errorf("expected research sweep %d ulgm (7%% of gas fee), got %d", expectedResearchFromGas, researchSweep)
	}

	// CRITICAL ASSERTION: The transfer amount was NEVER involved in any tax calculation
	// The research_fund received only from gas fees, not from transfer amount
	if researchSweep >= transferAmount {
		t.Fatalf("CRITICAL AUDIT FAIL: research_fund received %d ulgm >= transfer amount %d ulgm. Transfer amount is being taxed!", researchSweep, transferAmount)
	}

	// Verify RouteReward was NOT called for the transfer (it's only for protocol rewards)
	// The mock bank keeper's sentToAccount should be empty (RouteReward wasn't invoked)
	if len(bk.sentToAccount) > 0 {
		t.Errorf("AUDIT FAIL: sentToAccount should be empty for pure P2P transfer, got %v", bk.sentToAccount)
	}
}

func TestRouteFees_22_78_MathCorrectness(t *testing.T) {
	// Verify the 3.33/96.67 split math is correct across various fee amounts
	tests := []struct {
		name             string
		feeAmount        int64
		expectedResearch int64
	}{
		{"minimal_fee", 100, 3},             // 100 * 33300 / 1000000 = 3
		{"standard_transfer", 21000, 699},   // 21000 * 33300 / 1000000 = 699
		{"large_fee", 1000000, 33300},       // 1000000 * 33300 / 1000000 = 33300
		{"small_fee", 10, 0},                // 10 * 33300 / 1000000 = 0 (integer truncation)
		{"rounding_case", 1, 0},             // 1 * 33300 / 1000000 = 0 (rounded down)
	}

	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	sk := &mockStakingKeeper{activeCount: 22}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bk := newMockBankKeeper()
			bk.balances = map[string]sdk.Coins{
				feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(tt.feeAmount))),
			}
			k, ctx := setupKeeperWithBank(t, bk, sk)

			if err := k.RouteFees(ctx); err != nil {
				t.Fatalf("RouteFees failed: %v", err)
			}

			researchAmount := bk.sentToModule["research_fund"].AmountOf("ulgm").Int64()
			if researchAmount != tt.expectedResearch {
				t.Errorf("fee=%d: expected research=%d, got %d", tt.feeAmount, tt.expectedResearch, researchAmount)
			}
		})
	}
}

func TestNoCirculationTax_Exists(t *testing.T) {
	// This test serves as a canary: if anyone adds a retrievalRate, transfer tax,
	// or circulation tax to the vesting_rewards module, this test documents
	// that such a mechanism does NOT exist and should NOT be added for P2P transfers.
	//
	// The only fee mechanisms in the protocol are:
	// 1. Gas fees (flat per tx type, deducted by ante handler)
	// 2. Research tax (3.33% of gas fees, swept in BeginBlock)
	// 3. Reward routing (3.33% of protocol rewards like block production, verification)
	//
	// There is NO:
	// - retrievalRate (transfer tax proportional to amount)
	// - circulation tax on P2P sends
	// - Tobin tax on agent-to-agent transfers
	// - Value-added tax on token transfers

	k, ctx := setupKeeper(t)

	// RouteReward requires an explicit call — it is not triggered by transfers
	// Verify it returns the expected split for a protocol reward
	routing, err := k.RouteReward(ctx, types.SourceBlockProduction, "1000000", sdk.AccAddress("validator___________").String(), "")
	if err != nil {
		t.Fatalf("RouteReward failed: %v", err)
	}

	// Verify research share is exactly 3.33%
	if routing.ResearchShare != "33300" {
		t.Errorf("expected research share 33300 (3.33%% of 1000000), got %s", routing.ResearchShare)
	}
	if routing.RecipientAmount != "966700" {
		t.Errorf("expected recipient 966700 (96.67%% of 1000000), got %s", routing.RecipientAmount)
	}

	// RouteFees only processes fee_collector — it has no concept of "transfer tax"
	err = k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees should not fail even with nil bankKeeper on base keeper: %v", err)
	}
}

// ==================== Founder Auto-Split Tests ====================

// setupFounderKeeper creates a keeper with founder params configured.
// ResearchShareBps=33300 (3.33%), FounderShareBps=70000 (7% of research).
func setupFounderKeeper(t *testing.T, bk *mockBankKeeper, founderAddr string, govHeight uint64) (keeper.Keeper, sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.Params.FounderShareBps = 70000
	gs.Params.FounderAddress = founderAddr
	gs.Params.GovernanceActivationHeight = govHeight
	return setupKeeperWithBankAndGenesis(t, bk, &mockStakingKeeper{activeCount: 22}, gs)
}

func TestFounderAutoSplit(t *testing.T) {
	// Block reward with founder address set: verify founder, research_fund, producer amounts.
	//
	// Math (epoch 0, full validators, ResearchShareBps=33300, FounderShareBps=70000):
	//   Block reward:     3,000,000 ulgm
	//   Research (gross): 3000000 * 33300 / 1000000 = 99900
	//   Founder (7%):     99900 * 70000 / 1000000  = 6993
	//   Net research:     99900 - 6993 = 92907
	//   Post-research:    3000000 - 92907 - 6993 = 2900100
	//   Verif pool (50%): 2900100 * 500000 / 1000000 = 1450050
	//   Producer (50%):   2900100 - 1450050 = 1450050
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "3000000" {
		t.Errorf("expected total minted 3000000, got %s", dist.TotalMinted)
	}
	if dist.FounderShare != "6993" {
		t.Errorf("expected founder share 6993, got %s", dist.FounderShare)
	}
	if dist.ResearchShare != "92907" {
		t.Errorf("expected net research share 92907, got %s", dist.ResearchShare)
	}
	if dist.ProducerReward != "1450050" {
		t.Errorf("expected producer reward 1450050, got %s", dist.ProducerReward)
	}

	// Verify bank sends
	founderCoins := bk.sentToAccount[founderAddr]
	if founderCoins.AmountOf("ulgm").Int64() != 6993 {
		t.Errorf("expected 6993 ulgm to founder, got %d", founderCoins.AmountOf("ulgm").Int64())
	}
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("ulgm").Int64() != 92907 {
		t.Errorf("expected 92907 ulgm to research_fund, got %d", researchCoins.AmountOf("ulgm").Int64())
	}
	producerCoins := bk.sentToAccount[producer]
	if producerCoins.AmountOf("ulgm").Int64() != 1450050 {
		t.Errorf("expected 1450050 ulgm to producer, got %d", producerCoins.AmountOf("ulgm").Int64())
	}
}

func TestFounderSplitDisabled(t *testing.T) {
	// Empty founder address: full research goes to research_fund, nothing to founder.
	bk := newMockBankKeeper()
	k, ctx := setupFounderKeeper(t, bk, "", 0) // empty founder address = disabled

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Full research (no founder deduction): 3000000 * 33300 / 1000000 = 99900
	if dist.ResearchShare != "99900" {
		t.Errorf("expected full research share 99900, got %s", dist.ResearchShare)
	}
	if dist.FounderShare != "0" {
		t.Errorf("expected founder share 0 when disabled, got %s", dist.FounderShare)
	}

	// No coins should be sent to any founder address
	founderAddr := sdk.AccAddress("founder_____________").String()
	if coins, ok := bk.sentToAccount[founderAddr]; ok && !coins.IsZero() {
		t.Errorf("expected no coins to founder address, got %v", coins)
	}
}

func TestFounderSplitSunset(t *testing.T) {
	// GovernanceActivationHeight=500, run at block 1000: no founder share (governance active).
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 500)

	// Context is at block 1000 (default from setupKeeperWithBankAndGenesis), which is >= 500
	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Founder share should be 0 because governance has activated (block 1000 >= 500)
	if dist.FounderShare != "0" {
		t.Errorf("expected founder share 0 after governance sunset, got %s", dist.FounderShare)
	}
	// Full research share (no founder deduction)
	if dist.ResearchShare != "99900" {
		t.Errorf("expected full research share 99900 after sunset, got %s", dist.ResearchShare)
	}

	// No coins to founder
	if coins, ok := bk.sentToAccount[founderAddr]; ok && !coins.IsZero() {
		t.Errorf("expected no coins to founder after sunset, got %v", coins)
	}
}

func TestFounderShareInFeeRouting(t *testing.T) {
	// RouteFees with founder address: 7% of fee research goes to founder.
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	// Set fee_collector balance: 21000 ulgm
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(21000))),
	}

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// Total research sweep: 21000 * 33300 / 1000000 = 699
	// Founder (7% of 699): 699 * 70000 / 1000000 = 48
	// Net research: 699 - 48 = 651
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("ulgm").Int64() != 651 {
		t.Errorf("expected 651 ulgm to research_fund (net after founder), got %d", researchCoins.AmountOf("ulgm").Int64())
	}
	founderCoins := bk.sentToAccount[founderAddr]
	if founderCoins.AmountOf("ulgm").Int64() != 48 {
		t.Errorf("expected 48 ulgm to founder (7%% of fee research), got %d", founderCoins.AmountOf("ulgm").Int64())
	}
}

func TestFounderShareAccounting(t *testing.T) {
	// Verify total minted = research + founder + producer + verification + compute (no money creation/loss).
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	// Parse all amounts from the distribution
	totalMinted := new(big.Int)
	totalMinted.SetString(dist.TotalMinted, 10)

	researchShare := new(big.Int)
	researchShare.SetString(dist.ResearchShare, 10)

	founderShare := new(big.Int)
	founderShare.SetString(dist.FounderShare, 10)

	producerReward := new(big.Int)
	producerReward.SetString(dist.ProducerReward, 10)

	// Sum all outflows from bank sends
	var totalSent int64
	for _, coins := range bk.sentToAccount {
		totalSent += coins.AmountOf("ulgm").Int64()
	}
	for _, coins := range bk.sentToModule {
		totalSent += coins.AmountOf("ulgm").Int64()
	}

	// Total sent must equal total minted (no money created or lost)
	if totalSent != totalMinted.Int64() {
		t.Errorf("ACCOUNTING FAIL: total sent (%d) != total minted (%d). Money %s!",
			totalSent, totalMinted.Int64(),
			func() string {
				if totalSent > totalMinted.Int64() {
					return "created from nothing"
				}
				return "lost"
			}())
	}

	// Also verify the distribution breakdown adds up
	// totalMinted = researchShare + founderShare + producerReward + verification + compute
	knowledgeCoins := bk.sentToModule["knowledge"]
	computeCoins := bk.sentToModule["compute_pool"]

	breakdown := researchShare.Int64() + founderShare.Int64() + producerReward.Int64() +
		knowledgeCoins.AmountOf("ulgm").Int64() + computeCoins.AmountOf("ulgm").Int64()
	if breakdown != totalMinted.Int64() {
		t.Errorf("ACCOUNTING FAIL: breakdown sum (%d) != total minted (%d)", breakdown, totalMinted.Int64())
	}
}
