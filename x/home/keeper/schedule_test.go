package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

// ---------- Mock ScheduleKeeper ----------

type mockScheduleKeeper struct {
	tasks     map[string]mockTask // keyed by homeID+"/"+taskID
	nextSeq   int
	cancelled []string // taskIDs that were cancelled
}

type mockTask struct {
	homeID         string
	taskType       uint32
	scheduledHeight uint64
	intervalBlocks uint64
	taskID         string
}

func newMockScheduleKeeper() *mockScheduleKeeper {
	return &mockScheduleKeeper{
		tasks: make(map[string]mockTask),
	}
}

func (m *mockScheduleKeeper) RegisterTask(_ sdk.Context, homeID string, taskType uint32, scheduledHeight uint64, intervalBlocks uint64, _ []byte) (string, error) {
	m.nextSeq++
	prefix := homeID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	taskID := prefix + "-task-" + string(rune('0'+m.nextSeq))
	key := homeID + "/" + taskID
	m.tasks[key] = mockTask{
		homeID:          homeID,
		taskType:        taskType,
		scheduledHeight: scheduledHeight,
		intervalBlocks:  intervalBlocks,
		taskID:          taskID,
	}
	return taskID, nil
}

func (m *mockScheduleKeeper) CancelTask(_ sdk.Context, homeID, taskID string) bool {
	key := homeID + "/" + taskID
	if _, ok := m.tasks[key]; !ok {
		return false
	}
	delete(m.tasks, key)
	m.cancelled = append(m.cancelled, taskID)
	return true
}

// ---------- Tests ----------

func TestScheduleGuardianMaintenance(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a home
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Schedule Test Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	// Wire mock schedule keeper
	msk := newMockScheduleKeeper()
	k.SetScheduleKeeper(msk)

	// Schedule guardian maintenance
	err = k.ScheduleGuardianMaintenance(ctx, homeID)
	if err != nil {
		t.Fatalf("schedule guardian maintenance: %v", err)
	}

	// Verify taskID was stored
	taskID, exists := k.GetGuardianTaskID(ctx, homeID)
	if !exists {
		t.Fatal("expected guardian task ID to exist")
	}
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	// Verify mock received the correct parameters
	if len(msk.tasks) != 1 {
		t.Fatalf("expected 1 task registered, got %d", len(msk.tasks))
	}
	for _, task := range msk.tasks {
		if task.taskType != keeper.GuardianMaintenanceTaskType {
			t.Errorf("expected task type %d, got %d", keeper.GuardianMaintenanceTaskType, task.taskType)
		}
		if task.intervalBlocks != keeper.GuardianMaintenanceInterval {
			t.Errorf("expected interval %d, got %d", keeper.GuardianMaintenanceInterval, task.intervalBlocks)
		}
		expectedHeight := uint64(ctx.BlockHeight()) + keeper.GuardianMaintenanceInterval
		if task.scheduledHeight != expectedHeight {
			t.Errorf("expected scheduled height %d, got %d", expectedHeight, task.scheduledHeight)
		}
	}
}

func TestScheduleGuardianMaintenance_NilKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a home
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Nil Keeper Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Do NOT set schedule keeper — it remains nil
	err = k.ScheduleGuardianMaintenance(ctx, resp.HomeId)
	if err != nil {
		t.Fatalf("expected nil keeper to be a no-op, got: %v", err)
	}

	// No taskID should be stored
	_, exists := k.GetGuardianTaskID(ctx, resp.HomeId)
	if exists {
		t.Error("expected no guardian task ID when keeper is nil")
	}
}

func TestScheduleGuardianMaintenance_Idempotent(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Idempotent Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	msk := newMockScheduleKeeper()
	k.SetScheduleKeeper(msk)

	// Schedule once
	err = k.ScheduleGuardianMaintenance(ctx, homeID)
	if err != nil {
		t.Fatalf("first schedule: %v", err)
	}
	firstTaskID, _ := k.GetGuardianTaskID(ctx, homeID)

	// Schedule again — should cancel old and register new
	err = k.ScheduleGuardianMaintenance(ctx, homeID)
	if err != nil {
		t.Fatalf("second schedule: %v", err)
	}
	secondTaskID, exists := k.GetGuardianTaskID(ctx, homeID)
	if !exists {
		t.Fatal("expected task ID after re-schedule")
	}

	// The old task should have been cancelled
	if len(msk.cancelled) != 1 {
		t.Fatalf("expected 1 cancellation, got %d", len(msk.cancelled))
	}
	if msk.cancelled[0] != firstTaskID {
		t.Errorf("expected cancelled task %s, got %s", firstTaskID, msk.cancelled[0])
	}

	// New task should be different
	if firstTaskID == secondTaskID {
		t.Error("expected different task ID after re-schedule")
	}

	// Only 1 task should remain in mock
	if len(msk.tasks) != 1 {
		t.Errorf("expected 1 task remaining, got %d", len(msk.tasks))
	}
}

func TestCancelGuardianMaintenance(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Cancel Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	msk := newMockScheduleKeeper()
	k.SetScheduleKeeper(msk)

	// Schedule first
	err = k.ScheduleGuardianMaintenance(ctx, homeID)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}

	taskID, exists := k.GetGuardianTaskID(ctx, homeID)
	if !exists {
		t.Fatal("expected task ID before cancel")
	}

	// Cancel
	k.CancelGuardianMaintenance(ctx, homeID)

	// Task ID should be removed from store
	_, exists = k.GetGuardianTaskID(ctx, homeID)
	if exists {
		t.Error("expected guardian task ID to be removed after cancel")
	}

	// Mock should reflect cancellation
	if len(msk.cancelled) != 1 || msk.cancelled[0] != taskID {
		t.Errorf("expected mock cancellation of %s, got %v", taskID, msk.cancelled)
	}
	if len(msk.tasks) != 0 {
		t.Errorf("expected 0 tasks remaining, got %d", len(msk.tasks))
	}
}

func TestCancelGuardianMaintenance_NoTask(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "No Task Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	msk := newMockScheduleKeeper()
	k.SetScheduleKeeper(msk)

	// Cancel without scheduling — should be a no-op
	k.CancelGuardianMaintenance(ctx, resp.HomeId)

	if len(msk.cancelled) != 0 {
		t.Error("expected no cancellations for home without task")
	}
}

func TestScheduleGuardianMaintenance_HomeNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	msk := newMockScheduleKeeper()
	k.SetScheduleKeeper(msk)

	err := k.ScheduleGuardianMaintenance(ctx, "nonexistent-home")
	if err == nil {
		t.Fatal("expected error for nonexistent home")
	}
}
