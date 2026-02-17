package orchestrator

import "testing"

func TestMovementState_Lifecycle(t *testing.T) {
	spec := MovementSpec{
		Name: "test-movement",
		Parts: []PartSpec{
			{ID: "part-a", Section: SectionStrings, Action: ActionRegister},
			{ID: "part-b", Section: SectionBrass, Action: ActionSubmitClaim},
		},
		Gate:    GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	if ms.Status != MovementPending {
		t.Errorf("initial status = %s, want pending", ms.Status)
	}

	ms.Start(50)
	if ms.Status != MovementRunning {
		t.Errorf("status after start = %s, want running", ms.Status)
	}
	if ms.StartBlock != 50 {
		t.Errorf("start block = %d, want 50", ms.StartBlock)
	}
}

func TestMovementState_PartTracking(t *testing.T) {
	spec := MovementSpec{
		Name: "test",
		Parts: []PartSpec{
			{ID: "p1", Section: SectionStrings, Action: ActionRegister},
			{ID: "p2", Section: SectionStrings, Action: ActionStake},
		},
		Gate:    GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	ms.Start(1)

	// Assign and complete p1.
	if err := ms.RecordPartAssignment("p1", "agent-a"); err != nil {
		t.Fatalf("assign p1: %v", err)
	}
	if err := ms.CompletePart("p1", &PartResult{Success: true}); err != nil {
		t.Fatalf("complete p1: %v", err)
	}

	if ms.CompletedCount() != 1 {
		t.Errorf("completed = %d, want 1", ms.CompletedCount())
	}
	if ms.AllPartsCompleted() {
		t.Error("all parts completed before p2 done")
	}

	// Assign and complete p2.
	if err := ms.RecordPartAssignment("p2", "agent-b"); err != nil {
		t.Fatalf("assign p2: %v", err)
	}
	if err := ms.CompletePart("p2", &PartResult{Success: true}); err != nil {
		t.Fatalf("complete p2: %v", err)
	}

	if !ms.AllPartsCompleted() {
		t.Error("expected all parts completed")
	}
}

func TestMovementState_FailPart_RetryExhausted(t *testing.T) {
	spec := MovementSpec{
		Name:  "test",
		Parts: []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister, MaxRetries: 2}},
		Gate:  GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	ms.Start(1)

	// First attempt fails.
	if err := ms.RecordPartAssignment("p1", "agent-a"); err != nil {
		t.Fatalf("assign p1: %v", err)
	}
	retry, err := ms.FailPart("p1", &PartResult{Success: false, Error: "fail-1"})
	if err != nil {
		t.Fatalf("fail p1 attempt 1: %v", err)
	}
	if !retry {
		t.Error("expected retries remaining after first failure")
	}

	// Second attempt fails — should exhaust retries.
	if err := ms.RecordPartAssignment("p1", "agent-b"); err != nil {
		t.Fatalf("assign p1 retry: %v", err)
	}
	retry, err = ms.FailPart("p1", &PartResult{Success: false, Error: "fail-2"})
	if err != nil {
		t.Fatalf("fail p1 attempt 2: %v", err)
	}
	if retry {
		t.Error("expected no retries remaining")
	}

	if !ms.HasFailedParts() {
		t.Error("expected has_failed_parts after retry exhaustion")
	}
}

func TestMovementState_Timeout(t *testing.T) {
	spec := MovementSpec{
		Name:    "test",
		Parts:   []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister}},
		Gate:    GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	ms.Start(50)

	if ms.IsTimedOut(100) {
		t.Error("should not be timed out at block 100 (started 50, timeout 100)")
	}
	if !ms.IsTimedOut(200) {
		t.Error("should be timed out at block 200 (started 50, timeout 100)")
	}
}

func TestEvaluateGate_BlockHeight(t *testing.T) {
	ms := NewMovementState(MovementSpec{Name: "test", Timeout: 100})

	gate := GateSpec{Type: GateBlockHeight, BlockHeight: 500}
	result := EvaluateGate(gate, ms, 499, nil, nil)
	if result.Passed {
		t.Error("gate should fail at block 499")
	}

	result = EvaluateGate(gate, ms, 500, nil, nil)
	if !result.Passed {
		t.Error("gate should pass at block 500")
	}
}

func TestEvaluateGate_Balance(t *testing.T) {
	ms := NewMovementState(MovementSpec{Name: "test", Timeout: 100})

	gate := GateSpec{Type: GateBalance, MinBalance: 1000}

	result := EvaluateGate(gate, ms, 0, func() uint64 { return 999 }, nil)
	if result.Passed {
		t.Error("gate should fail with balance 999")
	}

	result = EvaluateGate(gate, ms, 0, func() uint64 { return 1000 }, nil)
	if !result.Passed {
		t.Error("gate should pass with balance 1000")
	}
}

func TestEvaluateGate_Completion(t *testing.T) {
	spec := MovementSpec{
		Name: "test",
		Parts: []PartSpec{
			{ID: "a", Section: SectionStrings, Action: ActionRegister},
			{ID: "b", Section: SectionStrings, Action: ActionRegister},
			{ID: "c", Section: SectionStrings, Action: ActionRegister},
		},
		Gate:    GateSpec{Type: GateCompletion, Count: 2},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	ms.Start(1)

	// Complete 1 part.
	ms.RecordPartAssignment("a", "agent-1")
	ms.CompletePart("a", &PartResult{Success: true})

	gate := GateSpec{Type: GateCompletion, Count: 2}
	result := EvaluateGate(gate, ms, 0, nil, nil)
	if result.Passed {
		t.Error("gate should fail with 1/2 completions")
	}

	// Complete 2nd part.
	ms.RecordPartAssignment("b", "agent-2")
	ms.CompletePart("b", &PartResult{Success: true})

	result = EvaluateGate(gate, ms, 0, nil, nil)
	if !result.Passed {
		t.Error("gate should pass with 2/2 completions")
	}
}

func TestEvaluateGate_AllParts(t *testing.T) {
	spec := MovementSpec{
		Name: "test",
		Parts: []PartSpec{
			{ID: "x", Section: SectionStrings, Action: ActionRegister},
			{ID: "y", Section: SectionStrings, Action: ActionRegister},
		},
		Gate:    GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	ms.Start(1)

	gate := GateSpec{Type: GateAllParts}
	result := EvaluateGate(gate, ms, 0, nil, nil)
	if result.Passed {
		t.Error("gate should fail with 0 completions")
	}

	ms.RecordPartAssignment("x", "a1")
	ms.CompletePart("x", &PartResult{Success: true})
	ms.RecordPartAssignment("y", "a2")
	ms.CompletePart("y", &PartResult{Success: true})

	result = EvaluateGate(gate, ms, 0, nil, nil)
	if !result.Passed {
		t.Error("gate should pass with all parts completed")
	}
}

func TestEvaluateGate_Custom(t *testing.T) {
	ms := NewMovementState(MovementSpec{Name: "test", Timeout: 100})

	called := false
	customs := map[string]func() bool{
		"check-something": func() bool { called = true; return true },
	}

	gate := GateSpec{Type: GateCustom, CustomName: "check-something"}
	result := EvaluateGate(gate, ms, 0, nil, customs)
	if !result.Passed || !called {
		t.Error("custom gate should pass and be called")
	}

	gate = GateSpec{Type: GateCustom, CustomName: "nonexistent"}
	result = EvaluateGate(gate, ms, 0, nil, customs)
	if result.Passed {
		t.Error("nonexistent custom gate should fail")
	}
}

func TestRecordPartAssignment_UnknownPart(t *testing.T) {
	spec := MovementSpec{
		Name:  "test",
		Parts: []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister}},
		Gate:  GateSpec{Type: GateAllParts},
		Timeout: 100,
	}

	ms := NewMovementState(spec)
	err := ms.RecordPartAssignment("nonexistent", "agent-a")
	if err == nil {
		t.Error("expected error for unknown part")
	}
}
