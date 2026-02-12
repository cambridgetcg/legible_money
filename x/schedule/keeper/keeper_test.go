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
		Condition: types.ScheduleCondition{
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
			Condition: types.ScheduleCondition{
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
		Condition: types.ScheduleCondition{
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
		Condition: types.ScheduleCondition{
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
		Condition: types.ScheduleCondition{
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

