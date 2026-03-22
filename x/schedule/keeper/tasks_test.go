package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/schedule/keeper"
)

// ---------- Task API Tests ----------

func TestRegisterGetTask(t *testing.T) {
	k, ctx := setupKeeper(t)

	taskID, err := k.RegisterTask(ctx, "home-abc", keeper.TaskTypeGuardianMaintenance, 200, 0, []byte("payload1"))
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}
	if taskID == "" {
		t.Fatal("expected non-empty taskID")
	}

	task, found := k.GetTask(ctx, "home-abc", taskID)
	if !found {
		t.Fatal("task not found after registration")
	}
	if task.HomeID != "home-abc" {
		t.Errorf("expected homeID home-abc, got %s", task.HomeID)
	}
	if task.TaskType != keeper.TaskTypeGuardianMaintenance {
		t.Errorf("expected task type %d, got %d", keeper.TaskTypeGuardianMaintenance, task.TaskType)
	}
	if task.ScheduledHeight != 200 {
		t.Errorf("expected scheduled height 200, got %d", task.ScheduledHeight)
	}
	if task.IntervalBlocks != 0 {
		t.Errorf("expected interval 0 (one-shot), got %d", task.IntervalBlocks)
	}
	if string(task.Payload) != "payload1" {
		t.Errorf("expected payload 'payload1', got '%s'", string(task.Payload))
	}
	if task.CreatedAt != 100 { // setupKeeper sets block height to 100
		t.Errorf("expected createdAt 100, got %d", task.CreatedAt)
	}
}

func TestListTasks(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register 3 tasks for the same home
	for _, h := range []uint64{200, 300, 400} {
		_, err := k.RegisterTask(ctx, "home-xyz", keeper.TaskTypeCustom, h, 0, nil)
		if err != nil {
			t.Fatalf("RegisterTask at height %d failed: %v", h, err)
		}
	}

	// Register 1 task for a different home
	_, err := k.RegisterTask(ctx, "home-other", keeper.TaskTypeCreditRenewal, 250, 10, nil)
	if err != nil {
		t.Fatalf("RegisterTask for home-other failed: %v", err)
	}

	tasks := k.ListTasks(ctx, "home-xyz")
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks for home-xyz, got %d", len(tasks))
	}

	otherTasks := k.ListTasks(ctx, "home-other")
	if len(otherTasks) != 1 {
		t.Errorf("expected 1 task for home-other, got %d", len(otherTasks))
	}

	// Non-existent home returns empty
	empty := k.ListTasks(ctx, "home-none")
	if len(empty) != 0 {
		t.Errorf("expected 0 tasks for home-none, got %d", len(empty))
	}
}

func TestCancelTask(t *testing.T) {
	k, ctx := setupKeeper(t)

	taskID, err := k.RegisterTask(ctx, "home-abc", keeper.TaskTypeCustom, 200, 0, nil)
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}

	// Cancel existing task
	ok := k.CancelTask(ctx, "home-abc", taskID)
	if !ok {
		t.Fatal("expected CancelTask to return true")
	}

	// Verify it's gone
	_, found := k.GetTask(ctx, "home-abc", taskID)
	if found {
		t.Fatal("expected task to be deleted after cancel")
	}

	// Cancel non-existent task
	ok = k.CancelTask(ctx, "home-abc", "nonexistent")
	if ok {
		t.Fatal("expected CancelTask to return false for nonexistent task")
	}
}

func TestProcessDueTasks_OneShot(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register one-shot task due at height 100 (current block)
	taskID, err := k.RegisterTask(ctx, "home-abc", keeper.TaskTypeGuardianMaintenance, 100, 0, []byte("oneshot"))
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}

	dueTasks := k.ProcessDueTasks(ctx, 100)
	if len(dueTasks) != 1 {
		t.Fatalf("expected 1 due task, got %d", len(dueTasks))
	}
	if dueTasks[0].TaskID != taskID {
		t.Errorf("expected taskID %s, got %s", taskID, dueTasks[0].TaskID)
	}
	if string(dueTasks[0].Payload) != "oneshot" {
		t.Errorf("expected payload 'oneshot', got '%s'", string(dueTasks[0].Payload))
	}

	// One-shot task should be deleted
	_, found := k.GetTask(ctx, "home-abc", taskID)
	if found {
		t.Fatal("expected one-shot task to be deleted after processing")
	}
}

func TestProcessDueTasks_Recurring(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register recurring task due at height 100 with interval 50
	taskID, err := k.RegisterTask(ctx, "home-abc", keeper.TaskTypeCreditRenewal, 100, 50, nil)
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}

	dueTasks := k.ProcessDueTasks(ctx, 100)
	if len(dueTasks) != 1 {
		t.Fatalf("expected 1 due task, got %d", len(dueTasks))
	}

	// Recurring task should still exist with updated scheduledHeight
	task, found := k.GetTask(ctx, "home-abc", taskID)
	if !found {
		t.Fatal("expected recurring task to still exist after processing")
	}
	if task.ScheduledHeight != 150 { // 100 + 50
		t.Errorf("expected rescheduled height 150, got %d", task.ScheduledHeight)
	}

	// Should NOT fire at height 120 (next fire is 150)
	dueTasks = k.ProcessDueTasks(ctx, 120)
	if len(dueTasks) != 0 {
		t.Errorf("expected 0 due tasks at height 120, got %d", len(dueTasks))
	}

	// Should fire again at height 150
	dueTasks = k.ProcessDueTasks(ctx, 150)
	if len(dueTasks) != 1 {
		t.Fatalf("expected 1 due task at height 150, got %d", len(dueTasks))
	}

	// Verify rescheduled to 200
	task, found = k.GetTask(ctx, "home-abc", taskID)
	if !found {
		t.Fatal("expected recurring task to persist")
	}
	if task.ScheduledHeight != 200 { // 150 + 50
		t.Errorf("expected rescheduled height 200, got %d", task.ScheduledHeight)
	}
}

func TestProcessDueTasks_NotYetDue(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register task due at height 500
	_, err := k.RegisterTask(ctx, "home-abc", keeper.TaskTypeCustom, 500, 0, nil)
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}

	// Current height is 100 — nothing should be due
	dueTasks := k.ProcessDueTasks(ctx, 100)
	if len(dueTasks) != 0 {
		t.Errorf("expected 0 due tasks, got %d", len(dueTasks))
	}

	// Task should still exist
	tasks := k.ListTasks(ctx, "home-abc")
	if len(tasks) != 1 {
		t.Errorf("expected task to still exist, got %d tasks", len(tasks))
	}
}
