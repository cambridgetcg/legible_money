package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

// Conductor is the central coordinator that reads a Score and directs agents
// organized into Sections through sequential Movements.
type Conductor struct {
	score    *Score
	swarm    *swarm.Swarm
	sections map[SectionName]*Section
	executor ActionExecutor
	store    StateStore
	poller   *ChainPoller

	state           ConductorState
	currentMovement int
	movements       []*MovementState
	startTime       time.Time
	lastCheckpoint  time.Time
	totalGasSpent   uint64
	totalRewards    uint64

	onEvent     EventHandler
	customGates map[string]func() bool

	pauseCh  chan struct{}
	resumeCh chan struct{}
	mu       sync.RWMutex
	logger   *log.Logger
}

// ConductorOption configures a conductor.
type ConductorOption func(*Conductor)

// WithEventHandler sets an event callback.
func WithEventHandler(h EventHandler) ConductorOption {
	return func(c *Conductor) { c.onEvent = h }
}

// WithStateStore sets the checkpoint store.
func WithStateStore(s StateStore) ConductorOption {
	return func(c *Conductor) { c.store = s }
}

// WithExecutor sets the action executor.
func WithExecutor(e ActionExecutor) ConductorOption {
	return func(c *Conductor) { c.executor = e }
}

// WithCustomGates registers named gate functions for GateCustom evaluation.
func WithCustomGates(gates map[string]func() bool) ConductorOption {
	return func(c *Conductor) { c.customGates = gates }
}

// WithLogger sets the logger.
func WithLogger(l *log.Logger) ConductorOption {
	return func(c *Conductor) { c.logger = l }
}

// NewConductor creates a conductor for the given score and swarm.
func NewConductor(score *Score, sw *swarm.Swarm, opts ...ConductorOption) (*Conductor, error) {
	if err := score.Validate(); err != nil {
		return nil, fmt.Errorf("invalid score: %w", err)
	}

	c := &Conductor{
		score:    score,
		swarm:    sw,
		sections: make(map[SectionName]*Section),
		state:    StateIdle,
		pauseCh:  make(chan struct{}, 1),
		resumeCh: make(chan struct{}, 1),
		logger:   log.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.executor == nil {
		c.executor = NewStandardExecutor(nil)
	}
	if c.store == nil {
		c.store = &MemoryStateStore{}
	}

	return c, nil
}

// State returns the current conductor state.
func (c *Conductor) State() ConductorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Score returns the score being performed.
func (c *Conductor) Score() *Score { return c.score }

// Sections returns a snapshot of the current sections.
func (c *Conductor) Sections() map[SectionName]*Section {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[SectionName]*Section, len(c.sections))
	for k, v := range c.sections {
		result[k] = v
	}
	return result
}

// Tune initializes the orchestra: organizes agents into sections, elects leaders,
// and prepares movement state. Must be called before Perform.
func (c *Conductor) Tune(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateIdle && c.state != StateAborted && c.state != StateFailed {
		c.mu.Unlock()
		return fmt.Errorf("cannot tune from state %s", c.state)
	}
	c.state = StateTuning
	c.mu.Unlock()

	c.emit(Event{Type: EventTuneStarted, Message: fmt.Sprintf("tuning %s", c.score.Name)})

	// Check for existing checkpoint and resume if available.
	if c.store.Exists() {
		cp, err := c.store.Load()
		if err == nil && cp.ScoreName == c.score.Name {
			c.currentMovement = cp.CurrentMovement
			c.totalGasSpent = cp.TotalGasSpent
			c.totalRewards = cp.TotalRewards
			c.logger.Printf("[conductor] resuming from checkpoint: movement %d", cp.CurrentMovement)
		}
	}

	// Organize agents into sections.
	for _, spec := range c.score.Ensemble.Sections {
		role := SectionRoleMap[spec.Name]
		agents := c.swarm.AgentsByRole(role)
		if len(agents) < spec.MinCount {
			c.mu.Lock()
			c.state = StateFailed
			c.mu.Unlock()
			return fmt.Errorf("section %s needs %d agents with role %s, swarm has %d",
				spec.Name, spec.MinCount, role, len(agents))
		}
		// Cap to MaxCount.
		if len(agents) > spec.MaxCount {
			agents = agents[:spec.MaxCount]
		}

		section, err := NewSection(spec, agents)
		if err != nil {
			c.mu.Lock()
			c.state = StateFailed
			c.mu.Unlock()
			return fmt.Errorf("create section %s: %w", spec.Name, err)
		}

		c.mu.Lock()
		c.sections[spec.Name] = section
		c.mu.Unlock()

		c.emit(Event{
			Type:    EventLeaderElected,
			Section: spec.Name,
			Agent:   section.Leader.Managed.Spec.Name,
			Message: fmt.Sprintf("section %s leader: %s", spec.Name, section.Leader.Managed.Spec.Name),
		})
	}

	// Prepare movement states.
	c.movements = make([]*MovementState, len(c.score.Movements))
	for i, ms := range c.score.Movements {
		c.movements[i] = NewMovementState(ms)
	}

	c.mu.Lock()
	c.state = StateIdle
	c.mu.Unlock()

	c.emit(Event{Type: EventTuneCompleted, Message: "tuning complete"})
	return nil
}

// Perform executes the score's movements sequentially. Blocks until complete,
// failed, or aborted. Call Tune() first.
func (c *Conductor) Perform(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateIdle {
		c.mu.Unlock()
		return fmt.Errorf("cannot perform from state %s (did you call Tune?)", c.state)
	}
	if len(c.sections) == 0 {
		c.mu.Unlock()
		return fmt.Errorf("no sections configured — call Tune() first")
	}
	c.state = StatePerforming
	c.startTime = time.Now()
	c.mu.Unlock()

	c.emit(Event{Type: EventPerformStarted, Message: fmt.Sprintf("performing %s", c.score.Name)})

	// Start chain poller if configured.
	if c.poller != nil {
		c.poller.Start(ctx)
		defer c.poller.Stop()
	}

	// Execute movements sequentially, starting from checkpoint.
	for i := c.currentMovement; i < len(c.movements); i++ {
		// Check for pause.
		if err := c.checkPause(ctx); err != nil {
			return err
		}

		c.mu.Lock()
		c.currentMovement = i
		c.mu.Unlock()

		ms := c.movements[i]

		c.emit(Event{
			Type:     EventMovementStarted,
			Movement: ms.Spec.Name,
			Message:  fmt.Sprintf("movement %d: %s", i+1, ms.Spec.Name),
		})

		blockHeight := int64(0)
		if c.poller != nil {
			blockHeight = c.poller.Height()
		}
		ms.Start(blockHeight)

		// Dispatch parts.
		if err := c.dispatchParts(ctx, ms); err != nil {
			c.mu.Lock()
			c.state = StateFailed
			c.mu.Unlock()
			c.emit(Event{Type: EventMovementFailed, Movement: ms.Spec.Name, Message: err.Error()})
			return fmt.Errorf("movement %q: %w", ms.Spec.Name, err)
		}

		// Evaluate gate.
		if err := c.waitForGate(ctx, ms); err != nil {
			c.mu.Lock()
			c.state = StateFailed
			c.mu.Unlock()
			c.emit(Event{Type: EventMovementFailed, Movement: ms.Spec.Name, Message: err.Error()})
			return fmt.Errorf("movement %q gate: %w", ms.Spec.Name, err)
		}

		ms.mu.Lock()
		ms.Status = MovementCompleted
		ms.mu.Unlock()

		c.emit(Event{
			Type:     EventMovementCompleted,
			Movement: ms.Spec.Name,
			Message:  fmt.Sprintf("movement %q complete", ms.Spec.Name),
		})

		// Checkpoint after each movement.
		c.saveCheckpoint(i + 1)
	}

	c.mu.Lock()
	c.state = StateComplete
	c.mu.Unlock()

	c.emit(Event{Type: EventPerformCompleted, Message: "performance complete"})
	return nil
}

// dispatchParts assigns all parts in a movement to section members and executes them.
func (c *Conductor) dispatchParts(ctx context.Context, ms *MovementState) error {
	for _, partSpec := range ms.Spec.Parts {
		section, ok := c.sections[partSpec.Section]
		if !ok {
			return fmt.Errorf("part %q references unknown section %s", partSpec.ID, partSpec.Section)
		}

		if partSpec.RepeatPerAgent {
			// Broadcast: every member executes this part.
			members := section.BroadcastPart()
			for _, member := range members {
				if err := c.executePart(ctx, ms, partSpec, member); err != nil {
					return err
				}
				if err := StaggerDelay(ctx, c.score.Tempo.StaggerDelay); err != nil {
					return err
				}
			}
		} else {
			// Assign to least-loaded member.
			if err := c.executeWithRetry(ctx, ms, partSpec, section); err != nil {
				return err
			}
		}
	}
	return nil
}

// executeWithRetry attempts part execution with section-level retry.
func (c *Conductor) executeWithRetry(ctx context.Context, ms *MovementState, partSpec PartSpec, section *Section) error {
	maxRetries := partSpec.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		member, err := section.AssignPart(partSpec.ID)
		if err != nil {
			return err
		}

		err = c.executePart(ctx, ms, partSpec, member)
		if err == nil {
			return nil
		}

		c.emit(Event{
			Type:    EventPartRetried,
			Section: section.Name,
			Agent:   member.Managed.Spec.Name,
			PartID:  partSpec.ID,
			Message: fmt.Sprintf("retry %d/%d: %s", attempt+1, maxRetries, err),
		})
	}
	return fmt.Errorf("part %q: max retries exceeded", partSpec.ID)
}

// executePart runs a single part on a specific member.
func (c *Conductor) executePart(ctx context.Context, ms *MovementState, partSpec PartSpec, member *SectionMember) error {
	agentName := member.Managed.Spec.Name

	if err := ms.RecordPartAssignment(partSpec.ID, agentName); err != nil {
		return err
	}
	member.ActiveParts++

	c.emit(Event{
		Type:    EventPartStarted,
		Section: partSpec.Section,
		Agent:   agentName,
		PartID:  partSpec.ID,
		Message: fmt.Sprintf("%s → %s: %s", partSpec.Section, agentName, partSpec.Action),
	})

	result, err := c.executor.Execute(ctx, member.Managed.Agent, partSpec.Action, partSpec.Params)
	member.ActiveParts--

	if err != nil {
		member.TotalFailed++
		failResult := &PartResult{Success: false, Error: err.Error()}
		_ = ms.CompletePart(partSpec.ID, failResult)
		c.emit(Event{
			Type:    EventPartFailed,
			Section: partSpec.Section,
			Agent:   agentName,
			PartID:  partSpec.ID,
			Message: fmt.Sprintf("execution error: %s", err),
		})
		return err
	}

	if !result.Success {
		member.TotalFailed++
		_, _ = ms.FailPart(partSpec.ID, result)
		c.emit(Event{
			Type:    EventPartFailed,
			Section: partSpec.Section,
			Agent:   agentName,
			PartID:  partSpec.ID,
			Message: fmt.Sprintf("part failed: %s", result.Error),
		})
		return fmt.Errorf("part %q failed: %s", partSpec.ID, result.Error)
	}

	member.TotalDone++
	if err := ms.CompletePart(partSpec.ID, result); err != nil {
		return err
	}

	c.emit(Event{
		Type:    EventPartCompleted,
		Section: partSpec.Section,
		Agent:   agentName,
		PartID:  partSpec.ID,
		Message: fmt.Sprintf("part %q completed by %s", partSpec.ID, agentName),
	})
	return nil
}

// waitForGate polls until the movement's gate passes or times out.
func (c *Conductor) waitForGate(ctx context.Context, ms *MovementState) error {
	gate := ms.Spec.Gate

	// For GateAllParts, if already complete, pass immediately.
	if gate.Type == GateAllParts && ms.AllPartsCompleted() {
		c.emit(Event{Type: EventGatePassed, Movement: ms.Spec.Name, Message: "all parts gate passed"})
		return nil
	}
	if gate.Type == GateCompletion && ms.CompletedCount() >= gate.Count {
		c.emit(Event{Type: EventGatePassed, Movement: ms.Spec.Name, Message: "completion gate passed"})
		return nil
	}

	pollInterval := c.score.Tempo.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentBlock := int64(0)
			if c.poller != nil {
				currentBlock = c.poller.Height()
			}

			// Check gate before timeout — gate passing takes priority.
			result := EvaluateGate(gate, ms, currentBlock, nil, c.customGates)
			if result.Passed {
				c.emit(Event{Type: EventGatePassed, Movement: ms.Spec.Name, Message: result.Reason})
				return nil
			}

			// Gate didn't pass — check timeout.
			if ms.IsTimedOut(currentBlock) {
				c.emit(Event{Type: EventGateFailed, Movement: ms.Spec.Name, Message: "timeout"})
				return fmt.Errorf("movement %q timed out after %d blocks", ms.Spec.Name, ms.Spec.Timeout)
			}
		}
	}
}

// Pause pauses the performance at the next safe point (between parts).
func (c *Conductor) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StatePerforming {
		return fmt.Errorf("cannot pause from state %s", c.state)
	}

	select {
	case c.pauseCh <- struct{}{}:
	default:
	}
	return nil
}

// Resume resumes a paused performance.
func (c *Conductor) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StatePaused {
		return fmt.Errorf("cannot resume from state %s", c.state)
	}

	c.state = StatePerforming
	select {
	case c.resumeCh <- struct{}{}:
	default:
	}

	c.emit(Event{Type: EventResumed, Message: "performance resumed"})
	return nil
}

// Abort stops the performance immediately.
func (c *Conductor) Abort() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = StateAborted
	c.saveCheckpointLocked(c.currentMovement)
	c.emit(Event{Type: EventAborted, Message: "performance aborted"})
}

// checkPause checks if a pause has been requested and waits for resume.
func (c *Conductor) checkPause(ctx context.Context) error {
	select {
	case <-c.pauseCh:
		c.mu.Lock()
		c.state = StatePaused
		c.mu.Unlock()

		c.saveCheckpoint(c.currentMovement)
		c.emit(Event{Type: EventPaused, Message: "performance paused"})

		// Wait for resume or cancel.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.resumeCh:
			return nil
		}
	default:
		return nil
	}
}

// saveCheckpoint persists current state. Called after each movement.
func (c *Conductor) saveCheckpoint(movementIdx int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.saveCheckpointLocked(movementIdx)
}

// saveCheckpointLocked persists state (caller must hold at least read lock).
func (c *Conductor) saveCheckpointLocked(movementIdx int) {
	leaders := make(map[string]string)
	for name, sec := range c.sections {
		if sec.Leader != nil {
			leaders[string(name)] = sec.Leader.Managed.Spec.Name
		}
	}

	completedParts := make(map[string]bool)
	for _, ms := range c.movements {
		for id, pt := range ms.Parts {
			if pt.Completed {
				completedParts[id] = true
			}
		}
	}

	blockHeight := int64(0)
	if c.poller != nil {
		blockHeight = c.poller.Height()
	}

	cp := &Checkpoint{
		ScoreName:       c.score.Name,
		State:           c.state,
		CurrentMovement: movementIdx,
		CompletedParts:  completedParts,
		SectionLeaders:  leaders,
		BlockHeight:     blockHeight,
		Timestamp:       time.Now(),
		TotalGasSpent:   c.totalGasSpent,
		TotalRewards:    c.totalRewards,
	}

	if err := c.store.Save(cp); err != nil {
		c.logger.Printf("[conductor] checkpoint save failed: %v", err)
	} else {
		c.lastCheckpoint = cp.Timestamp
		c.emit(Event{Type: EventCheckpointSaved, Message: fmt.Sprintf("checkpoint at movement %d", movementIdx)})
	}
}

// Health returns the current orchestra health snapshot.
func (c *Conductor) Health() OrchestraHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	h := OrchestraHealth{
		Score:          c.score.Name,
		State:          c.state,
		MovementsTotal: len(c.score.Movements),
		MovementsDone:  c.currentMovement,
		LastCheckpoint: c.lastCheckpoint,
	}

	if c.currentMovement < len(c.score.Movements) {
		h.CurrentMovement = c.score.Movements[c.currentMovement].Name
	}

	if !c.startTime.IsZero() {
		h.Uptime = time.Since(c.startTime)
	}

	for _, sec := range c.sections {
		h.Sections = append(h.Sections, sec.Health())
	}

	h.Economic = EconomicHealth{
		TotalGasSpent: c.totalGasSpent,
		TotalRewards:  c.totalRewards,
		NetPosition:   int64(c.totalRewards) - int64(c.totalGasSpent),
		Sustainable:   c.totalRewards >= c.totalGasSpent,
	}
	if c.score.Economics.MaxTotalGasSpend > c.totalGasSpent {
		h.Economic.BudgetRemaining = c.score.Economics.MaxTotalGasSpend - c.totalGasSpent
	}

	return h
}

// SetPoller sets the chain poller for block-height tracking.
func (c *Conductor) SetPoller(p *ChainPoller) {
	c.poller = p
}

// emit sends an event to the handler if one is registered.
func (c *Conductor) emit(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if c.onEvent != nil {
		c.onEvent(e)
	}
}
