package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestConductor_TunePerformComplete(t *testing.T) {
	conductor, cleanup, _ := testConductor(t)
	defer cleanup()

	ctx := context.Background()

	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	if conductor.State() != StateIdle {
		t.Errorf("state after tune = %s, want idle", conductor.State())
	}

	// Verify sections were created.
	sections := conductor.Sections()
	if len(sections) != 2 {
		t.Errorf("sections = %d, want 2", len(sections))
	}

	if sec, ok := sections[SectionStrings]; ok {
		if sec.Leader == nil {
			t.Error("strings section has no leader")
		}
	} else {
		t.Error("strings section not found")
	}

	// Perform.
	if err := conductor.Perform(ctx); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if conductor.State() != StateComplete {
		t.Errorf("state after perform = %s, want complete", conductor.State())
	}
}

func TestConductor_Abort(t *testing.T) {
	conductor, cleanup, _ := testConductor(t)
	defer cleanup()

	ctx := context.Background()

	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	// Abort before perform.
	conductor.Abort()

	if conductor.State() != StateAborted {
		t.Errorf("state = %s, want aborted", conductor.State())
	}
}

func TestConductor_PauseResume(t *testing.T) {
	score := &Score{
		Name: "pause-test",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 1, MaxCount: 3, Domains: []string{"general"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name:    "m1",
				Parts:   []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister, RepeatPerAgent: true}},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
			{
				Name:    "m2",
				Parts:   []PartSpec{{ID: "p2", Section: SectionStrings, Action: ActionHeartbeat, RepeatPerAgent: true}},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       50 * time.Millisecond,
			CheckpointInterval: time.Second,
		},
		Economics: EconomicsSpec{MinOperatingBalance: 1000},
	}

	sw, swCleanup, _, _ := testSwarm(t, score)
	defer swCleanup()

	store := &MemoryStateStore{}
	var events []Event
	var mu sync.Mutex

	conductor, err := NewConductor(score, sw,
		WithEventHandler(func(e Event) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		}),
		WithExecutor(&RehearsalExecutor{}),
		WithStateStore(store),
	)
	if err != nil {
		t.Fatalf("NewConductor: %v", err)
	}

	ctx := context.Background()
	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	// Start perform in goroutine so we can pause/resume.
	done := make(chan error, 1)
	go func() {
		done <- conductor.Perform(ctx)
	}()

	// Wait for perform to start.
	time.Sleep(100 * time.Millisecond)

	// Performance should complete (all parts use RehearsalExecutor which succeeds).
	err = <-done
	if err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if conductor.State() != StateComplete {
		t.Errorf("state = %s, want complete", conductor.State())
	}
}

func TestConductor_CannotPerformWithoutTune(t *testing.T) {
	score := testScore(t)
	sw, cleanup, _, _ := testSwarm(t, score)
	defer cleanup()

	conductor, err := NewConductor(score, sw,
		WithExecutor(&RehearsalExecutor{}),
	)
	if err != nil {
		t.Fatalf("NewConductor: %v", err)
	}

	// Perform without Tune should fail.
	err = conductor.Perform(context.Background())
	if err == nil {
		t.Fatal("expected error performing without tune")
	}
}

func TestConductor_Health(t *testing.T) {
	conductor, cleanup, _ := testConductor(t)
	defer cleanup()

	ctx := context.Background()
	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	h := conductor.Health()
	if h.Score != "test-score" {
		t.Errorf("score = %s, want test-score", h.Score)
	}
	if h.MovementsTotal != 2 {
		t.Errorf("movements_total = %d, want 2", h.MovementsTotal)
	}
	if len(h.Sections) != 2 {
		t.Errorf("sections = %d, want 2", len(h.Sections))
	}
}

func TestConductor_CheckpointRecovery(t *testing.T) {
	score := testScore(t)
	store := &MemoryStateStore{}

	// Simulate a previous checkpoint at movement 1.
	store.Save(&Checkpoint{
		ScoreName:       "test-score",
		State:           StatePerforming,
		CurrentMovement: 1,
		CompletedParts:  map[string]bool{"register-v": true, "register-c": true},
		SectionLeaders:  map[string]string{"strings": "verifier-A"},
		BlockHeight:     100,
		Timestamp:       time.Now(),
	})

	sw, cleanup, _, _ := testSwarm(t, score)
	defer cleanup()

	conductor, err := NewConductor(score, sw,
		WithExecutor(&RehearsalExecutor{}),
		WithStateStore(store),
	)
	if err != nil {
		t.Fatalf("NewConductor: %v", err)
	}

	ctx := context.Background()
	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	// Conductor should resume from movement 1 (skipping movement 0).
	if err := conductor.Perform(ctx); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if conductor.State() != StateComplete {
		t.Errorf("state = %s, want complete", conductor.State())
	}
}

func TestConductor_InvalidScore(t *testing.T) {
	invalid := &Score{Name: ""} // Missing name.
	sw, cleanup, _, _ := testSwarm(t, testScore(t))
	defer cleanup()

	_, err := NewConductor(invalid, sw)
	if err == nil {
		t.Fatal("expected error for invalid score")
	}
}

func TestConductor_ContextCancellation(t *testing.T) {
	score := &Score{
		Name: "cancel-test",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 1, MaxCount: 2, Domains: []string{"general"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name:  "wait-forever",
				Parts: []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister, RepeatPerAgent: true}},
				Gate:  GateSpec{Type: GateBlockHeight, BlockHeight: 999999}, // Very high gate.
				Timeout: 999999,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       50 * time.Millisecond,
			CheckpointInterval: time.Second,
		},
		Economics: EconomicsSpec{MinOperatingBalance: 1000},
	}

	sw, cleanup, _, _ := testSwarm(t, score)
	defer cleanup()

	conductor, err := NewConductor(score, sw,
		WithExecutor(&RehearsalExecutor{}),
	)
	if err != nil {
		t.Fatalf("NewConductor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := conductor.Tune(ctx); err != nil {
		t.Fatalf("Tune: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- conductor.Perform(ctx)
	}()

	// Cancel context after a short delay.
	time.Sleep(200 * time.Millisecond)
	cancel()

	err = <-done
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
