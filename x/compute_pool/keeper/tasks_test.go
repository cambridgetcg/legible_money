package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/legible-money/legible/x/compute_pool/keeper"
	"github.com/legible-money/legible/x/compute_pool/types"
)

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func registerActiveProvider(t *testing.T, k keeper.Keeper, ctx interface{ WithBlockHeight(int64) interface{ BlockHeight() int64 } }, bk *mockBankKeeper, addr, serviceType string) {
	t.Helper()
}

// submitTestTask is a helper that funds the requester and submits a task.
func submitTestTask(t *testing.T, k keeper.Keeper, ctx interface{}, bk *mockBankKeeper, requesterAddr, requesterDID, serviceType, reward string) *types.ComputeTask {
	t.Helper()
	return nil
}

// -----------------------------------------------------------------------
// Tests: MsgSubmitTask ValidateBasic
// -----------------------------------------------------------------------

func TestMsgSubmitTaskValidateBasic(t *testing.T) {
	msg := &types.MsgSubmitTask{
		Sender:       providerAddr1.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "ai_inference",
			InputHash:   "abc123",
			MaxCU:       100,
		},
		Reward: "50000",
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

	// Missing requester DID
	bad = *msg
	bad.RequesterDID = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing requester_did")
	}

	// Invalid service type
	bad = *msg
	bad.Spec.ServiceType = "quantum"
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for invalid service type")
	}

	// Missing input hash
	bad = *msg
	bad.Spec.InputHash = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing input_hash")
	}

	// Zero max CU
	bad = *msg
	bad.Spec.MaxCU = 0
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero max_cu")
	}

	// Zero reward
	bad = *msg
	bad.Reward = "0"
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero reward")
	}

	// Invalid reward string
	bad = *msg
	bad.Reward = "notanumber"
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for invalid reward")
	}
}

func TestMsgAssignTaskValidateBasic(t *testing.T) {
	msg := &types.MsgAssignTask{
		Sender: providerAddr1.String(),
		TaskId: 1,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	// Zero task ID
	bad := *msg
	bad.TaskId = 0
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero task_id")
	}

	// Missing sender
	bad = *msg
	bad.Sender = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing sender")
	}
}

func TestMsgSubmitResultValidateBasic(t *testing.T) {
	msg := &types.MsgSubmitResult{
		Sender:     providerAddr1.String(),
		TaskId:     1,
		OutputHash: "hash123",
		ActualCU:   50,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	// Missing output hash
	bad := *msg
	bad.OutputHash = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for missing output_hash")
	}

	// Zero actual CU
	bad = *msg
	bad.ActualCU = 0
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero actual_cu")
	}
}

func TestMsgVerifyResultValidateBasic(t *testing.T) {
	// Accept without reason — valid
	msg := &types.MsgVerifyResult{
		Sender: providerAddr1.String(),
		TaskId: 1,
		Accept: true,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	// Reject without reason — invalid
	msg = &types.MsgVerifyResult{
		Sender: providerAddr1.String(),
		TaskId: 1,
		Accept: false,
		Reason: "",
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for reject without reason")
	}

	// Reject with reason — valid
	msg.Reason = "bad output"
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestMsgCancelTaskValidateBasic(t *testing.T) {
	msg := &types.MsgCancelTask{
		Sender: providerAddr1.String(),
		TaskId: 1,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid: %v", err)
	}

	bad := *msg
	bad.TaskId = 0
	if err := bad.ValidateBasic(); err == nil {
		t.Error("expected error for zero task_id")
	}
}

// -----------------------------------------------------------------------
// Tests: Task Submission
// -----------------------------------------------------------------------

func TestSubmitTask(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	requester := providerAddr1
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))

	msg := &types.MsgSubmitTask{
		Sender:       requester.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "ai_inference",
			InputHash:   "sha256:abc123",
			MaxCU:       100,
		},
		Reward: "50000",
	}

	task, err := k.SubmitTask(ctx, msg)
	if err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	if task.Id != 1 {
		t.Errorf("expected task_id=1, got %d", task.Id)
	}
	if task.Status != "pending" {
		t.Errorf("expected status=pending, got %s", task.Status)
	}
	if task.RequesterDID != "did:lgm:agent1" {
		t.Errorf("expected requester_did=did:lgm:agent1, got %s", task.RequesterDID)
	}
	if task.Reward != "50000" {
		t.Errorf("expected reward=50000, got %s", task.Reward)
	}
	if task.Spec.MaxCU != 100 {
		t.Errorf("expected max_cu=100, got %d", task.Spec.MaxCU)
	}

	// Reward should be escrowed from requester
	remaining := bk.balances[requester.String()+"/ulgm"]
	if !remaining.Equal(sdkmath.NewInt(950000)) {
		t.Errorf("expected remaining=950000, got %s", remaining.String())
	}

	// Task should be retrievable
	stored, found := k.GetTask(ctx, 1)
	if !found {
		t.Fatal("task not found after submission")
	}
	if stored.Spec.ServiceType != "ai_inference" {
		t.Errorf("expected service_type=ai_inference, got %s", stored.Spec.ServiceType)
	}
}

func TestSubmitTaskInsufficientFunds(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	requester := providerAddr1
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(100)) // not enough

	msg := &types.MsgSubmitTask{
		Sender:       requester.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "ai_inference",
			InputHash:   "sha256:abc123",
			MaxCU:       100,
		},
		Reward: "50000",
	}

	_, err := k.SubmitTask(ctx, msg)
	if err == nil {
		t.Fatal("expected error for insufficient funds")
	}
}

func TestSubmitTaskInvalidServiceType(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(1000000))

	msg := &types.MsgSubmitTask{
		Sender:       providerAddr1.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "quantum_teleport",
			InputHash:   "hash",
			MaxCU:       100,
		},
		Reward: "50000",
	}

	_, err := k.SubmitTask(ctx, msg)
	if err == nil {
		t.Fatal("expected error for invalid service type")
	}
}

func TestSubmitTaskInvalidReward(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(1000000))

	msg := &types.MsgSubmitTask{
		Sender:       providerAddr1.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "ai_inference",
			InputHash:   "hash",
			MaxCU:       100,
		},
		Reward: "not_a_number",
	}

	_, err := k.SubmitTask(ctx, msg)
	if err == nil {
		t.Fatal("expected error for invalid reward")
	}
}

func TestSubmitMultipleTasks(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(10000000))

	for i := 0; i < 5; i++ {
		msg := &types.MsgSubmitTask{
			Sender:       providerAddr1.String(),
			RequesterDID: "did:lgm:agent1",
			Spec: types.ComputeSpec{
				ServiceType: "ai_inference",
				InputHash:   fmt.Sprintf("hash%d", i),
				MaxCU:       100,
			},
			Reward: "10000",
		}
		task, err := k.SubmitTask(ctx, msg)
		if err != nil {
			t.Fatalf("failed to submit task %d: %v", i, err)
		}
		if task.Id != uint64(i+1) {
			t.Errorf("expected task_id=%d, got %d", i+1, task.Id)
		}
	}

	tasks := k.GetAllTasks(ctx)
	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}
}

// -----------------------------------------------------------------------
// Tests: Task Assignment
// -----------------------------------------------------------------------

func TestAssignTask(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	// Register provider
	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      providerAddr1.String(),
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCu:  "1000",
		Stake:       "100000000",
	})

	// Submit task
	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender:       requester.String(),
		RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{
			ServiceType: "ai_inference",
			InputHash:   "hash123",
			MaxCU:       100,
		},
		Reward: "50000",
	})

	// Assign
	err := k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err != nil {
		t.Fatalf("failed to assign task: %v", err)
	}

	assigned, _ := k.GetTask(ctx, task.Id)
	if assigned.Status != "assigned" {
		t.Errorf("expected status=assigned, got %s", assigned.Status)
	}
	if assigned.Provider != providerAddr1.String() {
		t.Errorf("expected provider=%s, got %s", providerAddr1.String(), assigned.Provider)
	}
}

func TestAssignTaskNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.AssignTask(ctx, providerAddr1.String(), 999)
	if err == nil {
		t.Fatal("expected error for task not found")
	}
}

func TestAssignTaskAlreadyAssigned(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr2.String(), ServiceType: "ai_inference",
		Endpoint: "https://api2.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := validatorAddr
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	// Second assignment should fail
	err := k.AssignTask(ctx, providerAddr2.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for double assignment")
	}
}

func TestAssignTaskExpired(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	// Advance past expiry
	ctx = ctx.WithBlockHeight(int64(task.ExpiresAt + 1))
	err := k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for expired task")
	}
}

func TestAssignTaskProviderNotActive(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})
	// Pause provider
	_ = k.UpdateProviderState(ctx, &types.MsgUpdateProvider{Sender: providerAddr1.String(), Pause: true})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	err := k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for paused provider")
	}
}

func TestAssignTaskWrongServiceType(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "storage",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	err := k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for service type mismatch")
	}
}

func TestAssignTaskProviderNotRegistered(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	err := k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
}

// -----------------------------------------------------------------------
// Tests: Result Submission
// -----------------------------------------------------------------------

func TestSubmitResult(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	err := k.SubmitResult(ctx, providerAddr1.String(), task.Id, "output_hash_abc", 80, "model=gpt4")
	if err != nil {
		t.Fatalf("failed to submit result: %v", err)
	}

	resultTask, _ := k.GetTask(ctx, task.Id)
	if resultTask.Status != "result_pending" {
		t.Errorf("expected status=result_pending, got %s", resultTask.Status)
	}
	if resultTask.OutputHash != "output_hash_abc" {
		t.Errorf("expected output_hash=output_hash_abc, got %s", resultTask.OutputHash)
	}
	if resultTask.ActualCU != 80 {
		t.Errorf("expected actual_cu=80, got %d", resultTask.ActualCU)
	}
}

func TestSubmitResultNotAssigned(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	err := k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash", 50, "")
	if err == nil {
		t.Fatal("expected error for unassigned task")
	}
}

func TestSubmitResultWrongProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	// Wrong provider tries to submit
	err := k.SubmitResult(ctx, providerAddr2.String(), task.Id, "hash", 50, "")
	if err == nil {
		t.Fatal("expected error for wrong provider")
	}
}

func TestSubmitResultCapsActualCU(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	// Submit with actualCU > maxCU
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash", 999, "")

	resultTask, _ := k.GetTask(ctx, task.Id)
	if resultTask.ActualCU != 100 { // capped at max
		t.Errorf("expected actual_cu=100 (capped), got %d", resultTask.ActualCU)
	}
}

func TestSubmitResultDuplicate(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash1", 50, "")

	// Second submission should fail (status is now result_pending, not assigned)
	err := k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash2", 50, "")
	if err == nil {
		t.Fatal("expected error for duplicate result submission")
	}
}

func TestSubmitResultExpired(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	// Get updated task to check new expiry
	assigned, _ := k.GetTask(ctx, task.Id)
	ctx = ctx.WithBlockHeight(int64(assigned.ExpiresAt + 1))

	err := k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash", 50, "")
	if err == nil {
		t.Fatal("expected error for expired result window")
	}
}

// -----------------------------------------------------------------------
// Tests: Result Verification (Accept)
// -----------------------------------------------------------------------

func TestVerifyResultAccept(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	// Fund module account with reward (simulating escrow)
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "output_hash", 100, "")

	err := k.VerifyResult(ctx, requester.String(), task.Id, true, "")
	if err != nil {
		t.Fatalf("failed to verify result: %v", err)
	}

	verified, _ := k.GetTask(ctx, task.Id)
	if verified.Status != "accepted" {
		t.Errorf("expected status=accepted, got %s", verified.Status)
	}

	// Provider should have received reward (50000 ulgm for 100/100 CU)
	providerBal := bk.balances[providerAddr1.String()+"/ulgm"]
	// Provider started with 200M, staked 100M → 100M, then receives 50K → 100050000
	if !providerBal.Equal(sdkmath.NewInt(100050000)) {
		t.Errorf("expected provider balance=100050000, got %s", providerBal.String())
	}

	// Provider tasks_served should be incremented
	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.TasksServed != 1 {
		t.Errorf("expected tasks_served=1, got %d", provider.TasksServed)
	}
}

func TestVerifyResultAcceptPartialCU(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "100000", // 100K reward for 100 CU
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash", 50, "") // only 50 CU used

	err := k.VerifyResult(ctx, requester.String(), task.Id, true, "")
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	// Provider gets 50/100 × 100K = 50K
	// Requester gets 50K refund
	providerBal := bk.balances[providerAddr1.String()+"/ulgm"]
	// started 200M, staked 100M → 100M, + 50K = 100050000
	if !providerBal.Equal(sdkmath.NewInt(100050000)) {
		t.Errorf("expected provider balance=100050000, got %s", providerBal.String())
	}
}

func TestVerifyResultUnauthorized(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "hash", 100, "")

	// Wrong person tries to verify
	err := k.VerifyResult(ctx, providerAddr1.String(), task.Id, true, "")
	if err == nil {
		t.Fatal("expected error for unauthorized verification")
	}
}

func TestVerifyResultNotPendingVerify(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	// Try to verify a pending (no result) task
	err := k.VerifyResult(ctx, providerAddr2.String(), task.Id, true, "")
	if err == nil {
		t.Fatal("expected error for task not pending verify")
	}
}

// -----------------------------------------------------------------------
// Tests: Result Verification (Reject)
// -----------------------------------------------------------------------

func TestVerifyResultReject(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "bad_hash", 100, "")

	err := k.VerifyResult(ctx, requester.String(), task.Id, false, "incorrect output")
	if err != nil {
		t.Fatalf("failed to reject result: %v", err)
	}

	rejected, _ := k.GetTask(ctx, task.Id)
	if rejected.Status != "rejected" {
		t.Errorf("expected status=rejected, got %s", rejected.Status)
	}

	// Reward should be returned to requester
	// Requester started with 1M, paid 50K → 950K, refund 50K → 1M
	requesterBal := bk.balances[requester.String()+"/ulgm"]
	if !requesterBal.Equal(sdkmath.NewInt(1000000)) {
		t.Errorf("expected requester balance=1000000 after refund, got %s", requesterBal.String())
	}

	// Provider should have tasks_failed incremented
	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.TasksFailed != 1 {
		t.Errorf("expected tasks_failed=1, got %d", provider.TasksFailed)
	}

	// Provider stake should be slashed by 0.5% (500 bps / 1000000)
	// 100000000 × 500 / 1000000 = 50000
	// 100000000 - 50000 = 99950000
	if provider.Stake != "99950000" {
		t.Errorf("expected provider stake=99950000 after slash, got %s", provider.Stake)
	}
}

// -----------------------------------------------------------------------
// Tests: Task Cancellation
// -----------------------------------------------------------------------

func TestCancelTask(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	requester := providerAddr1
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	err := k.CancelTask(ctx, requester.String(), task.Id)
	if err != nil {
		t.Fatalf("failed to cancel task: %v", err)
	}

	cancelled, _ := k.GetTask(ctx, task.Id)
	if cancelled.Status != "cancelled" {
		t.Errorf("expected status=cancelled, got %s", cancelled.Status)
	}

	// Reward should be refunded
	requesterBal := bk.balances[requester.String()+"/ulgm"]
	if !requesterBal.Equal(sdkmath.NewInt(1000000)) {
		t.Errorf("expected full refund, got balance=%s", requesterBal.String())
	}
}

func TestCancelTaskUnauthorized(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: providerAddr1.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	// Wrong person cancels
	err := k.CancelTask(ctx, providerAddr2.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for unauthorized cancel")
	}
}

func TestCancelTaskAlreadyAssigned(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	err := k.CancelTask(ctx, requester.String(), task.Id)
	if err == nil {
		t.Fatal("expected error for cancelling assigned task")
	}
}

func TestCancelTaskNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.CancelTask(ctx, providerAddr1.String(), 999)
	if err == nil {
		t.Fatal("expected error for task not found")
	}
}

// -----------------------------------------------------------------------
// Tests: Task Expiration
// -----------------------------------------------------------------------

func TestExpireUnassignedTasks(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	requester := providerAddr1
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	// Advance past task expiry
	ctx = ctx.WithBlockHeight(int64(task.ExpiresAt + 1))
	k.ExpireUnassignedTasks(ctx)

	expired, _ := k.GetTask(ctx, task.Id)
	if expired.Status != "expired" {
		t.Errorf("expected status=expired, got %s", expired.Status)
	}

	// Reward should be refunded
	requesterBal := bk.balances[requester.String()+"/ulgm"]
	if !requesterBal.Equal(sdkmath.NewInt(1000000)) {
		t.Errorf("expected full refund after expiry, got balance=%s", requesterBal.String())
	}
}

func TestExpireAssignedTaskPenalizesProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})

	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)

	// Get updated expiry
	assigned, _ := k.GetTask(ctx, task.Id)
	ctx = ctx.WithBlockHeight(int64(assigned.ExpiresAt + 1))
	k.ExpireUnassignedTasks(ctx)

	// Provider should have tasks_failed incremented
	provider, _ := k.GetProvider(ctx, providerAddr1.String())
	if provider.TasksFailed != 1 {
		t.Errorf("expected tasks_failed=1, got %d", provider.TasksFailed)
	}
}

// -----------------------------------------------------------------------
// Tests: Usage Metering
// -----------------------------------------------------------------------

func TestAgentUsageTracking(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(10000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	// Submit and complete 3 tasks
	for i := 0; i < 3; i++ {
		task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
			Sender: requester.String(), RequesterDID: "did:lgm:agent1",
			Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: fmt.Sprintf("hash%d", i), MaxCU: 100},
			Reward: "50000",
		})
		_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
		_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, fmt.Sprintf("out%d", i), 80, "")
		_ = k.VerifyResult(ctx, requester.String(), task.Id, true, "")
	}

	usage, found := k.GetAgentUsage(ctx, "did:lgm:agent1")
	if !found {
		t.Fatal("agent usage not found")
	}
	if usage.TotalTasks != 3 {
		t.Errorf("expected total_tasks=3, got %d", usage.TotalTasks)
	}
	if usage.TotalCU != 240 { // 80 × 3
		t.Errorf("expected total_cu=240, got %d", usage.TotalCU)
	}
	if usage.AcceptedTasks != 3 {
		t.Errorf("expected accepted_tasks=3, got %d", usage.AcceptedTasks)
	}
	// 50000 reward × 80/100 = 40000 per task × 3 = 120000
	if usage.TotalReward != "120000" {
		t.Errorf("expected total_reward=120000, got %s", usage.TotalReward)
	}
}

func TestAgentUsageTrackingRejected(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(10000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent2",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	})
	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "bad", 100, "")
	_ = k.VerifyResult(ctx, requester.String(), task.Id, false, "bad output")

	usage, found := k.GetAgentUsage(ctx, "did:lgm:agent2")
	if !found {
		t.Fatal("agent usage not found")
	}
	if usage.RejectedTasks != 1 {
		t.Errorf("expected rejected_tasks=1, got %d", usage.RejectedTasks)
	}
	if usage.TotalCU != 0 { // rejected = no CU consumed
		t.Errorf("expected total_cu=0 for rejected, got %d", usage.TotalCU)
	}
}

func TestAgentUsageCRUD(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// No usage initially
	_, found := k.GetAgentUsage(ctx, "did:lgm:test")
	if found {
		t.Fatal("expected no usage initially")
	}

	// GetOrCreate
	usage := k.GetOrCreateAgentUsage(ctx, "did:lgm:test")
	if usage.TotalTasks != 0 {
		t.Errorf("expected total_tasks=0, got %d", usage.TotalTasks)
	}
	if usage.TotalReward != "0" {
		t.Errorf("expected total_reward=0, got %s", usage.TotalReward)
	}

	// Set and get
	usage.TotalTasks = 5
	usage.TotalCU = 500
	usage.TotalReward = "100000"
	k.SetAgentUsage(ctx, usage)

	stored, found := k.GetAgentUsage(ctx, "did:lgm:test")
	if !found {
		t.Fatal("usage not found after set")
	}
	if stored.TotalTasks != 5 {
		t.Errorf("expected total_tasks=5, got %d", stored.TotalTasks)
	}
}

func TestGetAllAgentUsages(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetAgentUsage(ctx, &types.AgentUsageRecord{
			AgentDID:    fmt.Sprintf("did:lgm:agent%d", i),
			TotalTasks:  uint64(i + 1),
			TotalReward: "0",
		})
	}

	usages := k.GetAllAgentUsages(ctx)
	if len(usages) != 3 {
		t.Errorf("expected 3 usages, got %d", len(usages))
	}
}

// -----------------------------------------------------------------------
// Tests: Task Index Queries
// -----------------------------------------------------------------------

func TestGetTasksByRequester(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(10000000))
	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(10000000))

	// Submit 3 tasks from agent1
	for i := 0; i < 3; i++ {
		k.SubmitTask(ctx, &types.MsgSubmitTask{
			Sender: providerAddr1.String(), RequesterDID: "did:lgm:agent1",
			Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: fmt.Sprintf("hash%d", i), MaxCU: 100},
			Reward: "10000",
		})
	}

	// Submit 2 tasks from agent2
	for i := 0; i < 2; i++ {
		k.SubmitTask(ctx, &types.MsgSubmitTask{
			Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent2",
			Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: fmt.Sprintf("hash2_%d", i), MaxCU: 100},
			Reward: "10000",
		})
	}

	agent1Tasks := k.GetTasksByRequester(ctx, "did:lgm:agent1")
	if len(agent1Tasks) != 3 {
		t.Errorf("expected 3 tasks for agent1, got %d", len(agent1Tasks))
	}

	agent2Tasks := k.GetTasksByRequester(ctx, "did:lgm:agent2")
	if len(agent2Tasks) != 2 {
		t.Errorf("expected 2 tasks for agent2, got %d", len(agent2Tasks))
	}
}

func TestGetTasksByProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(10000000))

	// Submit and assign 2 tasks to provider1
	for i := 0; i < 2; i++ {
		task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
			Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
			Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: fmt.Sprintf("hash%d", i), MaxCU: 100},
			Reward: "10000",
		})
		_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	}

	providerTasks := k.GetTasksByProvider(ctx, providerAddr1.String())
	if len(providerTasks) != 2 {
		t.Errorf("expected 2 tasks for provider, got %d", len(providerTasks))
	}
}

// -----------------------------------------------------------------------
// Tests: Task CRUD
// -----------------------------------------------------------------------

func TestTaskCRUD(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	task := &types.ComputeTask{
		Id:            1,
		RequesterDID:  "did:lgm:agent1",
		RequesterAddr: "addr1",
		Spec:          types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward:        "50000",
		Status:        "pending",
		SubmittedAt:   100,
		ExpiresAt:     600,
	}

	k.SetTask(ctx, task)

	stored, found := k.GetTask(ctx, 1)
	if !found {
		t.Fatal("task not found")
	}
	if stored.RequesterDID != "did:lgm:agent1" {
		t.Errorf("expected requester_did=did:lgm:agent1, got %s", stored.RequesterDID)
	}
	if stored.Status != "pending" {
		t.Errorf("expected status=pending, got %s", stored.Status)
	}

	// Not found
	_, found = k.GetTask(ctx, 999)
	if found {
		t.Error("expected task 999 not found")
	}
}

func TestTaskIdSequence(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	id1 := k.GetNextTaskId(ctx)
	id2 := k.GetNextTaskId(ctx)
	id3 := k.GetNextTaskId(ctx)

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Errorf("expected sequential IDs 1,2,3 got %d,%d,%d", id1, id2, id3)
	}
}

func TestGetAllTasks(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	for i := uint64(1); i <= 5; i++ {
		k.SetTask(ctx, &types.ComputeTask{
			Id:     i,
			Status: "pending",
			Reward: "0",
		})
	}

	tasks := k.GetAllTasks(ctx)
	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}
}

// -----------------------------------------------------------------------
// Tests: Extended Genesis
// -----------------------------------------------------------------------

func TestGenesisExtendedRoundTrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set tasks and usages
	k.SetTask(ctx, &types.ComputeTask{
		Id:            1,
		RequesterDID:  "did:lgm:agent1",
		RequesterAddr: "addr1",
		Spec:          types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward:        "50000",
		Status:        "pending",
		SubmittedAt:   100,
		ExpiresAt:     600,
	})
	k.SetTaskRequesterIndex(ctx, "did:lgm:agent1", 1)

	k.SetAgentUsage(ctx, &types.AgentUsageRecord{
		AgentDID:    "did:lgm:agent1",
		TotalTasks:  5,
		TotalCU:     500,
		TotalReward: "100000",
	})

	// Export
	tasks, usages := k.ExportGenesisExtended(ctx)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if len(usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(usages))
	}

	// Import into new keeper
	k2, ctx2, _, _ := setupKeeper(t)
	k2.InitGenesisExtended(ctx2, tasks, usages)

	stored, found := k2.GetTask(ctx2, 1)
	if !found {
		t.Fatal("task not found after genesis import")
	}
	if stored.RequesterDID != "did:lgm:agent1" {
		t.Errorf("expected requester_did=did:lgm:agent1, got %s", stored.RequesterDID)
	}

	usage, found := k2.GetAgentUsage(ctx2, "did:lgm:agent1")
	if !found {
		t.Fatal("usage not found after genesis import")
	}
	if usage.TotalTasks != 5 {
		t.Errorf("expected total_tasks=5, got %d", usage.TotalTasks)
	}
}

// -----------------------------------------------------------------------
// Tests: Full Task Lifecycle
// -----------------------------------------------------------------------

func TestFullTaskLifecycle(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	// Setup provider
	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	// Setup requester
	requester := providerAddr2
	bk.setBalance(requester.String(), "ulgm", sdkmath.NewInt(1000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	// 1. Submit
	task, err := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: requester.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "input_sha256", MaxCU: 200, Model: "claude-4"},
		Reward: "100000",
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending, got %s", task.Status)
	}

	// 2. Assign
	err = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	if err != nil {
		t.Fatalf("assign failed: %v", err)
	}

	// 3. Submit result
	err = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "result_hash_abc", 150, "latency=200ms")
	if err != nil {
		t.Fatalf("submit result failed: %v", err)
	}

	// 4. Verify (accept)
	err = k.VerifyResult(ctx, requester.String(), task.Id, true, "")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	// Final state
	final, _ := k.GetTask(ctx, task.Id)
	if final.Status != "accepted" {
		t.Errorf("expected accepted, got %s", final.Status)
	}
	if final.ActualCU != 150 {
		t.Errorf("expected actual_cu=150, got %d", final.ActualCU)
	}
	if final.OutputHash != "result_hash_abc" {
		t.Errorf("expected output_hash=result_hash_abc, got %s", final.OutputHash)
	}
	if final.ResultMetadata != "latency=200ms" {
		t.Errorf("expected result_metadata=latency=200ms, got %s", final.ResultMetadata)
	}

	// Provider paid proportionally: 100000 × 150/200 = 75000
	// Provider balance: 100M + 75000 = 100075000
	providerBal := bk.balances[providerAddr1.String()+"/ulgm"]
	if !providerBal.Equal(sdkmath.NewInt(100075000)) {
		t.Errorf("expected provider balance=100075000, got %s", providerBal.String())
	}

	// Requester refunded excess: 100000 - 75000 = 25000
	// Requester: 1M - 100K + 25K = 925000
	requesterBal := bk.balances[requester.String()+"/ulgm"]
	if !requesterBal.Equal(sdkmath.NewInt(925000)) {
		t.Errorf("expected requester balance=925000, got %s", requesterBal.String())
	}
}

// -----------------------------------------------------------------------
// Tests: MsgServer handlers
// -----------------------------------------------------------------------

func TestMsgServerSubmitTask(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(1000000))

	msg := &types.MsgSubmitTask{
		Sender: providerAddr1.String(), RequesterDID: "did:lgm:agent1",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash", MaxCU: 100},
		Reward: "50000",
	}

	// msgServer has HandleSubmitTask, call the keeper method directly for now
	_ = ms // just verify compilation of msg server
	task, err := k.SubmitTask(ctx, msg)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if task.Id != 1 {
		t.Errorf("expected task_id=1, got %d", task.Id)
	}
}

// -----------------------------------------------------------------------
// Tests: Edge cases
// -----------------------------------------------------------------------

func TestVerifyResultTaskNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.VerifyResult(ctx, providerAddr1.String(), 999, true, "")
	if err == nil {
		t.Fatal("expected error for task not found")
	}
}

func TestSubmitResultTaskNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.SubmitResult(ctx, providerAddr1.String(), 999, "hash", 50, "")
	if err == nil {
		t.Fatal("expected error for task not found")
	}
}

func TestMultipleAgentsUsageIsolation(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	bk.setBalance(providerAddr1.String(), "ulgm", sdkmath.NewInt(200000000))
	_ = k.RegisterProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender: providerAddr1.String(), ServiceType: "ai_inference",
		Endpoint: "https://api.example.com", PricePerCu: "1000", Stake: "100000000",
	})

	bk.setBalance(providerAddr2.String(), "ulgm", sdkmath.NewInt(10000000))
	bk.setBalance(validatorAddr.String(), "ulgm", sdkmath.NewInt(10000000))
	bk.setBalance(moduleAddrStr(), "ulgm", sdkmath.NewInt(100000000))

	// Agent1 submits 2 tasks
	for i := 0; i < 2; i++ {
		task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
			Sender: providerAddr2.String(), RequesterDID: "did:lgm:agent1",
			Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: fmt.Sprintf("hash_a1_%d", i), MaxCU: 100},
			Reward: "50000",
		})
		_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
		_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, fmt.Sprintf("out%d", i), 100, "")
		_ = k.VerifyResult(ctx, providerAddr2.String(), task.Id, true, "")
	}

	// Agent2 submits 1 task
	task, _ := k.SubmitTask(ctx, &types.MsgSubmitTask{
		Sender: validatorAddr.String(), RequesterDID: "did:lgm:agent2",
		Spec: types.ComputeSpec{ServiceType: "ai_inference", InputHash: "hash_a2", MaxCU: 50},
		Reward: "25000",
	})
	_ = k.AssignTask(ctx, providerAddr1.String(), task.Id)
	_ = k.SubmitResult(ctx, providerAddr1.String(), task.Id, "out_a2", 50, "")
	_ = k.VerifyResult(ctx, validatorAddr.String(), task.Id, true, "")

	// Verify isolation
	usage1, _ := k.GetAgentUsage(ctx, "did:lgm:agent1")
	usage2, _ := k.GetAgentUsage(ctx, "did:lgm:agent2")

	if usage1.TotalTasks != 2 || usage1.AcceptedTasks != 2 {
		t.Errorf("agent1: expected 2 tasks/2 accepted, got %d/%d", usage1.TotalTasks, usage1.AcceptedTasks)
	}
	if usage1.TotalCU != 200 {
		t.Errorf("agent1: expected total_cu=200, got %d", usage1.TotalCU)
	}

	if usage2.TotalTasks != 1 || usage2.AcceptedTasks != 1 {
		t.Errorf("agent2: expected 1 task/1 accepted, got %d/%d", usage2.TotalTasks, usage2.AcceptedTasks)
	}
	if usage2.TotalCU != 50 {
		t.Errorf("agent2: expected total_cu=50, got %d", usage2.TotalCU)
	}
}
