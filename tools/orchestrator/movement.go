package orchestrator

import (
	"fmt"
	"sync"
)

// MovementStatus tracks execution state of a movement.
type MovementStatus string

const (
	MovementPending    MovementStatus = "pending"
	MovementRunning    MovementStatus = "running"
	MovementGateCheck  MovementStatus = "gate_check"
	MovementCompleted  MovementStatus = "completed"
	MovementFailed     MovementStatus = "failed"
)

// MovementState tracks the runtime state of a single movement.
type MovementState struct {
	Spec       MovementSpec
	Status     MovementStatus
	StartBlock int64 // Chain height when movement began
	Parts      map[string]*PartTracker
	mu         sync.RWMutex
}

// PartTracker tracks execution of a single part across assignments.
type PartTracker struct {
	Spec        PartSpec
	Assignments []PartAssignment // All assignments (including retries)
	Retries     int              // Total retries across all assignments
	Completed   bool
	Failed      bool
}

// GateResult captures the outcome of evaluating a movement's gate.
type GateResult struct {
	Passed  bool
	Reason  string
	Current interface{} // The measured value (block height, balance, count, etc.)
	Target  interface{} // The required value
}

// NewMovementState creates a movement state from a spec.
func NewMovementState(spec MovementSpec) *MovementState {
	parts := make(map[string]*PartTracker, len(spec.Parts))
	for _, ps := range spec.Parts {
		parts[ps.ID] = &PartTracker{Spec: ps}
	}
	return &MovementState{
		Spec:   spec,
		Status: MovementPending,
		Parts:  parts,
	}
}

// Start marks the movement as running and records the start block.
func (m *MovementState) Start(blockHeight int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Status = MovementRunning
	m.StartBlock = blockHeight
}

// RecordPartAssignment records an agent assignment for a part.
func (m *MovementState) RecordPartAssignment(partID, agentName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pt, ok := m.Parts[partID]
	if !ok {
		return fmt.Errorf("unknown part: %s", partID)
	}

	pt.Assignments = append(pt.Assignments, PartAssignment{
		PartID: partID,
		Agent:  agentName,
		Status: PartRunning,
	})
	return nil
}

// CompletePart marks a part's latest assignment as completed.
func (m *MovementState) CompletePart(partID string, result *PartResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pt, ok := m.Parts[partID]
	if !ok {
		return fmt.Errorf("unknown part: %s", partID)
	}

	if len(pt.Assignments) == 0 {
		return fmt.Errorf("part %s has no assignments", partID)
	}

	last := &pt.Assignments[len(pt.Assignments)-1]
	last.Status = PartCompleted
	last.Result = result
	pt.Completed = true
	return nil
}

// FailPart marks a part's latest assignment as failed and returns whether
// retries remain (based on the part spec's MaxRetries).
func (m *MovementState) FailPart(partID string, result *PartResult) (retriesRemain bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pt, ok := m.Parts[partID]
	if !ok {
		return false, fmt.Errorf("unknown part: %s", partID)
	}

	if len(pt.Assignments) == 0 {
		return false, fmt.Errorf("part %s has no assignments", partID)
	}

	last := &pt.Assignments[len(pt.Assignments)-1]
	last.Status = PartFailed
	last.Result = result
	pt.Retries++

	maxRetries := pt.Spec.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // default
	}

	if pt.Retries >= maxRetries {
		pt.Failed = true
		return false, nil
	}
	return true, nil
}

// CompletedCount returns the number of completed parts.
func (m *MovementState) CompletedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n := 0
	for _, pt := range m.Parts {
		if pt.Completed {
			n++
		}
	}
	return n
}

// AllPartsCompleted returns true if every part is completed.
func (m *MovementState) AllPartsCompleted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pt := range m.Parts {
		if !pt.Completed {
			return false
		}
	}
	return true
}

// HasFailedParts returns true if any part has exhausted retries.
func (m *MovementState) HasFailedParts() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pt := range m.Parts {
		if pt.Failed {
			return true
		}
	}
	return false
}

// IsTimedOut checks whether the movement has exceeded its timeout.
func (m *MovementState) IsTimedOut(currentBlock int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.StartBlock == 0 || m.Spec.Timeout <= 0 {
		return false
	}
	return currentBlock-m.StartBlock > m.Spec.Timeout
}

// EvaluateGate checks whether the movement's gate condition is met.
// The customGates map holds named functions for GateCustom type.
func EvaluateGate(gate GateSpec, ms *MovementState, currentBlock int64, balanceProvider func() uint64, customGates map[string]func() bool) GateResult {
	switch gate.Type {
	case GateBlockHeight:
		return GateResult{
			Passed:  currentBlock >= gate.BlockHeight,
			Reason:  fmt.Sprintf("block height: %d / %d", currentBlock, gate.BlockHeight),
			Current: currentBlock,
			Target:  gate.BlockHeight,
		}

	case GateBalance:
		bal := uint64(0)
		if balanceProvider != nil {
			bal = balanceProvider()
		}
		return GateResult{
			Passed:  bal >= gate.MinBalance,
			Reason:  fmt.Sprintf("balance: %d / %d ulgm", bal, gate.MinBalance),
			Current: bal,
			Target:  gate.MinBalance,
		}

	case GateCompletion:
		completed := ms.CompletedCount()
		return GateResult{
			Passed:  completed >= gate.Count,
			Reason:  fmt.Sprintf("completed: %d / %d", completed, gate.Count),
			Current: completed,
			Target:  gate.Count,
		}

	case GateAllParts:
		all := ms.AllPartsCompleted()
		return GateResult{
			Passed:  all,
			Reason:  fmt.Sprintf("all parts completed: %v", all),
			Current: ms.CompletedCount(),
			Target:  len(ms.Parts),
		}

	case GateCustom:
		if customGates == nil {
			return GateResult{Passed: false, Reason: "no custom gates registered"}
		}
		fn, ok := customGates[gate.CustomName]
		if !ok {
			return GateResult{Passed: false, Reason: fmt.Sprintf("unknown custom gate: %s", gate.CustomName)}
		}
		passed := fn()
		return GateResult{
			Passed: passed,
			Reason: fmt.Sprintf("custom gate %q: %v", gate.CustomName, passed),
		}

	default:
		return GateResult{Passed: false, Reason: fmt.Sprintf("unknown gate type: %s", gate.Type)}
	}
}
