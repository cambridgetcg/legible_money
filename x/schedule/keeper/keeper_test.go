package keeper_test

import (
	"context"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/schedule/keeper"
	"github.com/legible-money/legible/x/schedule/types"
)

// testCreator is a valid bech32 address for use in tests (set in init after prefix config).
var testCreator string

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.Seal()

	testCreator = sdk.AccAddress("schedule_creator____").String()
}

// ---------- Mock Keepers ----------

type mockBankKeeper struct {
	balances map[string]int64 // simplified balance tracking
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{balances: map[string]int64{
		testCreator: 1000000000, // 1000 LGM
	}}
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil // accept all transfers in tests
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		newMockBankKeeper(),
		testCreator,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

// ---------- Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.MaxActivePerAccount != 100 {
		t.Errorf("expected MaxActivePerAccount=100, got %d", params.MaxActivePerAccount)
	}

	params.MaxActivePerAccount = 50
	k.SetParams(ctx, params)

	updated := k.GetParams(ctx)
	if updated.MaxActivePerAccount != 50 {
		t.Errorf("expected MaxActivePerAccount=50, got %d", updated.MaxActivePerAccount)
	}
}

func TestSequence(t *testing.T) {
	k, ctx := setupKeeper(t)

	seq := k.GetSequence(ctx)
	if seq != 0 {
		t.Errorf("expected initial sequence 0, got %d", seq)
	}

	next := k.NextSequence(ctx)
	if next != 1 {
		t.Errorf("expected next sequence 1, got %d", next)
	}

	next = k.NextSequence(ctx)
	if next != 2 {
		t.Errorf("expected next sequence 2, got %d", next)
	}
}

func TestSetGetProcess(t *testing.T) {
	k, ctx := setupKeeper(t)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100000",
		CreatedAtBlock:  100,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}

	k.SetProcess(ctx, process)

	fetched, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found")
	}
	if fetched.Id != "sched-1" {
		t.Errorf("expected id sched-1, got %s", fetched.Id)
	}
	if fetched.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", fetched.Status)
	}
	if fetched.RemainingFee != "1000000" {
		t.Errorf("expected remaining fee 1000000, got %s", fetched.RemainingFee)
	}
}

func TestGetProcessesByCreator(t *testing.T) {
	k, ctx := setupKeeper(t)

	creator := testCreator

	for i := 1; i <= 3; i++ {
		k.SetProcess(ctx, types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-%d", i),
			Creator:         creator,
			Status:          types.StatusActive,
			RemainingFee:    "1000000",
			FeePerExecution: "100000",
			CreatedAtBlock:  100,
			NextExecuteAt:   200,
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{
					Type:           types.TimePeriodic,
					IntervalBlocks: 10,
				},
			},
		})
	}

	processes := k.GetProcessesByCreator(ctx, creator)
	if len(processes) != 3 {
		t.Errorf("expected 3 processes, got %d", len(processes))
	}
}

func TestCreateScheduleHandler(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	msg := &types.MsgCreateSchedule{
		Creator:         testCreator,
		FeePerExecution: "1000",
		PrepaidFee:      "100000", // 100 executions worth
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}

	resp, err := msgServer.CreateSchedule(ctx, msg)
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	if resp.ProcessId == "" {
		t.Fatal("expected process ID in response")
	}

	process, found := k.GetProcess(ctx, resp.ProcessId)
	if !found {
		t.Fatal("created process not found in store")
	}
	if process.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", process.Status)
	}
	if process.NextExecuteAt != 200 {
		t.Errorf("expected next_execute_at 200, got %d", process.NextExecuteAt)
	}
}

func TestProcessDueSchedules(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create a process due at block 100 (current height)
	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		MaxExecutions:   1,
		CreatedAtBlock:  50,
		NextExecuteAt:   100,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 100,
			},
		},
	}
	k.SetProcess(ctx, process)

	executed := k.ProcessDueSchedules(ctx)
	if executed != 1 {
		t.Errorf("expected 1 executed, got %d", executed)
	}

	// Verify process is now completed
	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found after execution")
	}
	if updated.Status != types.StatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
	if updated.ExecutionCount != 1 {
		t.Errorf("expected execution count 1, got %d", updated.ExecutionCount)
	}
}

func TestCancelScheduleRefund(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	creator := testCreator

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         creator,
		Status:          types.StatusActive,
		RemainingFee:    "500000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	resp, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   creator,
		ProcessId: "sched-1",
	})
	if err != nil {
		t.Fatalf("CancelSchedule failed: %v", err)
	}
	if resp.RefundedAmount != "500000" {
		t.Errorf("expected refund 500000, got %s", resp.RefundedAmount)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("cancelled process not found")
	}
	if updated.Status != types.StatusCancelled {
		t.Errorf("expected status cancelled, got %s", updated.Status)
	}
}

// ---------- Pause / Resume Lifecycle ----------

func TestPauseSchedule(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err != nil {
		t.Fatalf("PauseSchedule failed: %v", err)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("paused process not found")
	}
	if updated.Status != types.StatusPaused {
		t.Errorf("expected status paused, got %s", updated.Status)
	}
}

func TestPauseScheduleNotActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusCompleted,
		RemainingFee:    "0",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err == nil {
		t.Fatal("expected error pausing completed process")
	}
}

func TestPauseScheduleUnauthorized(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	otherAddr := sdk.AccAddress("other_address_______").String()

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   otherAddr,
		ProcessId: "sched-1",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestResumeSchedule(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusPaused,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err != nil {
		t.Fatalf("ResumeSchedule failed: %v", err)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("resumed process not found")
	}
	if updated.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", updated.Status)
	}
	if updated.NextExecuteAt == 0 {
		t.Error("expected non-zero NextExecuteAt after resume")
	}
}

func TestResumeScheduleNotPaused(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err == nil {
		t.Fatal("expected error resuming active process")
	}
}

func TestPauseResumeLifecycle(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	// Pause
	_, err := msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err != nil {
		t.Fatalf("PauseSchedule failed: %v", err)
	}

	p, _ := k.GetProcess(ctx, "sched-1")
	if p.Status != types.StatusPaused {
		t.Fatalf("expected paused, got %s", p.Status)
	}

	// Resume
	_, err = msgServer.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err != nil {
		t.Fatalf("ResumeSchedule failed: %v", err)
	}

	p, _ = k.GetProcess(ctx, "sched-1")
	if p.Status != types.StatusActive {
		t.Errorf("expected active after resume, got %s", p.Status)
	}
}

// ---------- AddFee ----------

func TestAddFee(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "100000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.AddFee(ctx, &types.MsgAddFee{
		Creator:   testCreator,
		ProcessId: "sched-1",
		Amount:    "50000",
	})
	if err != nil {
		t.Fatalf("AddFee failed: %v", err)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found after AddFee")
	}
	if updated.RemainingFee != "150000" {
		t.Errorf("expected remaining fee 150000, got %s", updated.RemainingFee)
	}
}

func TestAddFeeReactivatesExhausted(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusExhausted,
		RemainingFee:    "0",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   0,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.AddFee(ctx, &types.MsgAddFee{
		Creator:   testCreator,
		ProcessId: "sched-1",
		Amount:    "50000",
	})
	if err != nil {
		t.Fatalf("AddFee failed: %v", err)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found")
	}
	if updated.Status != types.StatusActive {
		t.Errorf("expected reactivated to active, got %s", updated.Status)
	}
	if updated.RemainingFee != "50000" {
		t.Errorf("expected remaining fee 50000, got %s", updated.RemainingFee)
	}
	if updated.NextExecuteAt == 0 {
		t.Error("expected NextExecuteAt recalculated after reactivation")
	}
}

func TestAddFeeUnauthorized(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	otherAddr := sdk.AccAddress("other_address_______").String()

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "100000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.AddFee(ctx, &types.MsgAddFee{
		Creator:   otherAddr,
		ProcessId: "sched-1",
		Amount:    "50000",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- UpdateParams ----------

func TestUpdateParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	newParams := types.DefaultParams()
	newParams.MaxActivePerAccount = 200

	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testCreator, // testCreator is the authority in setupKeeper
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	params := k.GetParams(ctx)
	if params.MaxActivePerAccount != 200 {
		t.Errorf("expected MaxActivePerAccount=200, got %d", params.MaxActivePerAccount)
	}
}

func TestUpdateParamsUnauthorized(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	otherAddr := sdk.AccAddress("other_address_______").String()

	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: otherAddr,
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected unauthorized error for non-authority sender")
	}
}

// ---------- ProcessDueSchedules Edge Cases ----------

func TestProcessDueSchedulesExhausted(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Process with fee less than fee-per-execution
	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "50", // less than feePerExecution
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   100, // due now
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 100,
			},
		},
	}
	k.SetProcess(ctx, process)

	executed := k.ProcessDueSchedules(ctx)
	if executed != 0 {
		t.Errorf("expected 0 executed (insufficient fee), got %d", executed)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found")
	}
	if updated.Status != types.StatusExhausted {
		t.Errorf("expected status exhausted, got %s", updated.Status)
	}
}

func TestProcessDueSchedulesPeriodic(t *testing.T) {
	k, ctx := setupKeeper(t)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		MaxExecutions:   0, // unlimited
		CreatedAtBlock:  50,
		NextExecuteAt:   100,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	executed := k.ProcessDueSchedules(ctx)
	if executed != 1 {
		t.Errorf("expected 1 executed, got %d", executed)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found after execution")
	}
	if updated.Status != types.StatusActive {
		t.Errorf("expected status active (periodic continues), got %s", updated.Status)
	}
	if updated.ExecutionCount != 1 {
		t.Errorf("expected execution count 1, got %d", updated.ExecutionCount)
	}
	// Periodic: next execution should be current height + interval
	if updated.NextExecuteAt != 110 {
		t.Errorf("expected next execution at 110, got %d", updated.NextExecuteAt)
	}
}

func TestProcessDueSchedulesExpired(t *testing.T) {
	k, ctx := setupKeeper(t)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		MaxExecutions:   0,
		ExpiresAtBlock:  100, // expires at current height
		CreatedAtBlock:  50,
		NextExecuteAt:   100,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	executed := k.ProcessDueSchedules(ctx)
	if executed != 1 {
		t.Errorf("expected 1 executed, got %d", executed)
	}

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("process not found")
	}
	if updated.Status != types.StatusCompleted {
		t.Errorf("expected status completed (expired), got %s", updated.Status)
	}
}

func TestProcessDueSchedulesNotDue(t *testing.T) {
	k, ctx := setupKeeper(t)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   500, // far future
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 500,
			},
		},
	}
	k.SetProcess(ctx, process)

	executed := k.ProcessDueSchedules(ctx)
	if executed != 0 {
		t.Errorf("expected 0 executed (not due), got %d", executed)
	}
}

// ---------- CleanupExpiredProcesses ----------

func TestCleanupExpiredProcesses(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create a completed process from long ago
	process := types.ScheduleProcess{
		Id:              "sched-old",
		Creator:         testCreator,
		Status:          types.StatusCompleted,
		RemainingFee:    "0",
		FeePerExecution: "100",
		ExecutionCount:  1,
		CreatedAtBlock:  1, // very old
	}
	k.SetProcess(ctx, process)

	// Advance context to high block height
	ctx = ctx.WithBlockHeight(1000000)
	cleaned := k.CleanupExpiredProcesses(ctx)
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	_, found := k.GetProcess(ctx, "sched-old")
	if found {
		t.Error("expected old process to be cleaned up")
	}
}

func TestCleanupSkipsActive(t *testing.T) {
	k, ctx := setupKeeper(t)

	process := types.ScheduleProcess{
		Id:              "sched-active",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  1,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	ctx = ctx.WithBlockHeight(1000000)
	cleaned := k.CleanupExpiredProcesses(ctx)
	if cleaned != 0 {
		t.Errorf("expected 0 cleaned (active process), got %d", cleaned)
	}

	_, found := k.GetProcess(ctx, "sched-active")
	if !found {
		t.Error("expected active process to still exist")
	}
}

// ---------- gRPC Queries ----------

func TestQueryParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	resp, err := queryServer.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("QueryParams failed: %v", err)
	}
	if resp.Params.MaxActivePerAccount != 100 {
		t.Errorf("expected MaxActivePerAccount=100, got %d", resp.Params.MaxActivePerAccount)
	}
}

func TestQueryProcess(t *testing.T) {
	k, ctx := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  100,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	resp, err := queryServer.Process(ctx, &types.QueryProcessRequest{ProcessId: "sched-1"})
	if err != nil {
		t.Fatalf("QueryProcess failed: %v", err)
	}
	if resp.Process.Id != "sched-1" {
		t.Errorf("expected id sched-1, got %s", resp.Process.Id)
	}
	if resp.Process.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", resp.Process.Status)
	}
}

func TestQueryProcessNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	_, err := queryServer.Process(ctx, &types.QueryProcessRequest{ProcessId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}
}

func TestQueryProcessesByCreator(t *testing.T) {
	k, ctx := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	for i := 1; i <= 3; i++ {
		k.SetProcess(ctx, types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-%d", i),
			Creator:         testCreator,
			Status:          types.StatusActive,
			RemainingFee:    "1000000",
			FeePerExecution: "100",
			CreatedAtBlock:  100,
			NextExecuteAt:   200,
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{
					Type:           types.TimeAbsolute,
					ExecuteAtBlock: 200,
				},
			},
		})
	}

	resp, err := queryServer.ProcessesByCreator(ctx, &types.QueryProcessesByCreatorRequest{Creator: testCreator})
	if err != nil {
		t.Fatalf("QueryProcessesByCreator failed: %v", err)
	}
	if len(resp.Processes) != 3 {
		t.Errorf("expected 3 processes, got %d", len(resp.Processes))
	}
}

// ---------- Genesis ----------

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create some state
	params := types.DefaultParams()
	params.MaxActivePerAccount = 50
	k.SetParams(ctx, params)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "500000",
		FeePerExecution: "100",
		CreatedAtBlock:  100,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)
	k.SetSequence(ctx, 5)

	// Export
	exported := k.ExportGenesis(ctx)
	if exported.Params == nil {
		t.Fatal("expected params in exported genesis")
	}
	if exported.Params.MaxActivePerAccount != 50 {
		t.Errorf("expected MaxActivePerAccount=50 in export, got %d", exported.Params.MaxActivePerAccount)
	}
	if len(exported.Processes) != 1 {
		t.Errorf("expected 1 process in export, got %d", len(exported.Processes))
	}
	if exported.Sequence != 5 {
		t.Errorf("expected sequence=5, got %d", exported.Sequence)
	}

	// Re-import into fresh keeper
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *exported)

	// Verify restored state
	restoredParams := k2.GetParams(ctx2)
	if restoredParams.MaxActivePerAccount != 50 {
		t.Errorf("expected restored MaxActivePerAccount=50, got %d", restoredParams.MaxActivePerAccount)
	}

	restoredProcess, found := k2.GetProcess(ctx2, "sched-1")
	if !found {
		t.Fatal("restored process not found")
	}
	if restoredProcess.RemainingFee != "500000" {
		t.Errorf("expected restored remaining fee 500000, got %s", restoredProcess.RemainingFee)
	}

	restoredSeq := k2.GetSequence(ctx2)
	if restoredSeq != 5 {
		t.Errorf("expected restored sequence=5, got %d", restoredSeq)
	}
}

// ---------- CancelSchedule Edge Cases ----------

func TestCancelScheduleAlreadyCancelled(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusCancelled,
		RemainingFee:    "0",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err == nil {
		t.Fatal("expected error cancelling already-cancelled process")
	}
}

func TestCancelScheduleAlreadyCompleted(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testCreator,
		Status:          types.StatusCompleted,
		RemainingFee:    "0",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
	}
	k.SetProcess(ctx, process)

	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   testCreator,
		ProcessId: "sched-1",
	})
	if err == nil {
		t.Fatal("expected error cancelling completed process")
	}
}

func TestCancelScheduleNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   testCreator,
		ProcessId: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}
}

// ---------- CreateSchedule Edge Cases ----------

func TestCreateScheduleMaxActiveExceeded(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set max to 2
	params := types.DefaultParams()
	params.MaxActivePerAccount = 2
	k.SetParams(ctx, params)

	// Create 2 active processes
	for i := 1; i <= 2; i++ {
		k.SetProcess(ctx, types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-%d", i),
			Creator:         testCreator,
			Status:          types.StatusActive,
			RemainingFee:    "1000000",
			FeePerExecution: "100",
			CreatedAtBlock:  100,
			NextExecuteAt:   200,
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{
					Type:           types.TimeAbsolute,
					ExecuteAtBlock: 200,
				},
			},
		})
	}

	// Third should fail
	_, err := msgServer.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator:         testCreator,
		FeePerExecution: "1000",
		PrepaidFee:      "100000",
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 300,
			},
		},
	})
	if err == nil {
		t.Fatal("expected max active schedules exceeded error")
	}
}

func TestCreateScheduleInvalidCondition(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator:         testCreator,
		FeePerExecution: "1000",
		PrepaidFee:      "100000",
		Condition:       nil, // nil condition
	})
	if err == nil {
		t.Fatal("expected invalid condition error")
	}
}

// ---------- CountActiveByCreator ----------

func TestCountActiveByCreator(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Mix of statuses
	statuses := []types.ProcessStatus{
		types.StatusActive,
		types.StatusPaused,
		types.StatusCompleted,
		types.StatusActive,
		types.StatusCancelled,
	}

	for i, status := range statuses {
		k.SetProcess(ctx, types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-%d", i),
			Creator:         testCreator,
			Status:          status,
			RemainingFee:    "1000000",
			FeePerExecution: "100",
			CreatedAtBlock:  100,
		})
	}

	// Active + Paused count: 2 active + 1 paused = 3
	count := k.CountActiveByCreator(ctx, testCreator)
	if count != 3 {
		t.Errorf("expected 3 active/paused, got %d", count)
	}
}

// ---------- Full Lifecycle ----------

func TestFullScheduleLifecycle(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// 1. Create
	resp, err := msgServer.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator:         testCreator,
		FeePerExecution: "1000",
		PrepaidFee:      "100000",
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	processId := resp.ProcessId

	// 2. Pause
	_, err = msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   testCreator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("PauseSchedule failed: %v", err)
	}

	// 3. Resume
	_, err = msgServer.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   testCreator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("ResumeSchedule failed: %v", err)
	}

	// 4. AddFee
	_, err = msgServer.AddFee(ctx, &types.MsgAddFee{
		Creator:   testCreator,
		ProcessId: processId,
		Amount:    "50000",
	})
	if err != nil {
		t.Fatalf("AddFee failed: %v", err)
	}

	p, found := k.GetProcess(ctx, processId)
	if !found {
		t.Fatal("process not found")
	}
	if p.RemainingFee != "150000" {
		t.Errorf("expected remaining fee 150000 (100000+50000), got %s", p.RemainingFee)
	}

	// 5. Cancel
	cancelResp, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   testCreator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("CancelSchedule failed: %v", err)
	}
	if cancelResp.RefundedAmount != "150000" {
		t.Errorf("expected refund 150000, got %s", cancelResp.RefundedAmount)
	}

	p, _ = k.GetProcess(ctx, processId)
	if p.Status != types.StatusCancelled {
		t.Errorf("expected final status cancelled, got %s", p.Status)
	}
}
