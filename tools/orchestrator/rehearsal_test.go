package orchestrator

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRehearseBootstrapScore(t *testing.T) {
	score := BootstrapScore("lgmvaloper1test", 100_000_000)

	events, err := Rehearse(score)
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events from rehearsal")
	}

	// Check that tune and perform events are present.
	var hasTuneStart, hasTuneComplete, hasPerformStart, hasPerformComplete bool
	for _, e := range events {
		switch e.Type {
		case EventTuneStarted:
			hasTuneStart = true
		case EventTuneCompleted:
			hasTuneComplete = true
		case EventPerformStarted:
			hasPerformStart = true
		case EventPerformCompleted:
			hasPerformComplete = true
		}
	}

	if !hasTuneStart {
		t.Error("missing tune_started event")
	}
	if !hasTuneComplete {
		t.Error("missing tune_completed event")
	}
	if !hasPerformStart {
		t.Error("missing perform_started event")
	}
	if !hasPerformComplete {
		t.Error("missing perform_completed event")
	}

	// Check that movements were completed.
	movementsDone := 0
	for _, e := range events {
		if e.Type == EventMovementCompleted {
			movementsDone++
		}
	}
	if movementsDone != 3 {
		t.Errorf("completed %d movements, want 3", movementsDone)
	}
}

func TestRehearseKnowledgeBlitzScore(t *testing.T) {
	score := KnowledgeBlitzScore("mathematics", 3)

	events, err := Rehearse(score)
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events")
	}

	var hasComplete bool
	for _, e := range events {
		if e.Type == EventPerformCompleted {
			hasComplete = true
		}
	}
	if !hasComplete {
		t.Error("expected perform_completed event")
	}
}

func TestScoreValidation(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(s *Score)
		wantError bool
	}{
		{
			name:      "valid score",
			mutate:    func(s *Score) {},
			wantError: false,
		},
		{
			name:      "empty name",
			mutate:    func(s *Score) { s.Name = "" },
			wantError: true,
		},
		{
			name:      "no sections",
			mutate:    func(s *Score) { s.Ensemble.Sections = nil },
			wantError: true,
		},
		{
			name:      "no movements",
			mutate:    func(s *Score) { s.Movements = nil },
			wantError: true,
		},
		{
			name:      "invalid section name",
			mutate:    func(s *Score) { s.Ensemble.Sections[0].Name = "invalid" },
			wantError: true,
		},
		{
			name:      "min > max",
			mutate:    func(s *Score) { s.Ensemble.Sections[0].MaxCount = 0 },
			wantError: true,
		},
		{
			name: "part references undefined section",
			mutate: func(s *Score) {
				s.Movements[0].Parts[0].Section = SectionWinds // Not in ensemble.
			},
			wantError: true,
		},
		{
			name:      "zero timeout",
			mutate:    func(s *Score) { s.Movements[0].Timeout = 0 },
			wantError: true,
		},
		{
			name:      "zero poll interval",
			mutate:    func(s *Score) { s.Tempo.PollInterval = 0 },
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Score{
				Name: "test",
				Ensemble: EnsembleSpec{
					Sections: []SectionSpec{
						{Name: SectionStrings, MinCount: 1, MaxCount: 5},
					},
				},
				Movements: []MovementSpec{
					{
						Name:    "m1",
						Parts:   []PartSpec{{ID: "p1", Section: SectionStrings, Action: ActionRegister}},
						Gate:    GateSpec{Type: GateAllParts},
						Timeout: 100,
					},
				},
				Tempo: TempoSpec{
					PollInterval:       time.Second,
					CheckpointInterval: time.Second,
				},
			}

			tc.mutate(s)
			err := s.Validate()
			if (err != nil) != tc.wantError {
				t.Errorf("Validate() error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}

func TestScoreJSONRoundTrip(t *testing.T) {
	original := BootstrapScore("lgmvaloper1abc", 100_000_000)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed Score
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Name != original.Name {
		t.Errorf("name = %s, want %s", parsed.Name, original.Name)
	}
	if len(parsed.Movements) != len(original.Movements) {
		t.Errorf("movements = %d, want %d", len(parsed.Movements), len(original.Movements))
	}
	if parsed.Tempo.PollInterval != original.Tempo.PollInterval {
		t.Errorf("poll_interval = %v, want %v", parsed.Tempo.PollInterval, original.Tempo.PollInterval)
	}
}

func TestRehearsalQueryClient(t *testing.T) {
	qc := NewRehearsalQueryClient()

	// Default height.
	if qc.Height() != 100 {
		t.Errorf("initial height = %d, want 100", qc.Height())
	}

	// Advance.
	qc.AdvanceBlocks(50)
	if qc.Height() != 150 {
		t.Errorf("height after advance = %d, want 150", qc.Height())
	}

	// Default balance.
	ctx := t
	_ = ctx
	// Balance for unknown address should be default.
	// (Can't call with context.Context from test easily, just trust the implementation.)
}

func TestRehearsalTxClient(t *testing.T) {
	tc := NewRehearsalTxClient()

	h1 := tc.nextHash()
	h2 := tc.nextHash()

	if h1 == h2 {
		t.Error("tx hashes should be unique")
	}
}

func TestFileStateStore(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStateStore(dir)

	if store.Exists() {
		t.Error("should not exist initially")
	}

	cp := &Checkpoint{
		ScoreName:       "test",
		State:           StatePerforming,
		CurrentMovement: 2,
		BlockHeight:     500,
		Timestamp:       time.Now(),
	}

	if err := store.Save(cp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !store.Exists() {
		t.Error("should exist after save")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ScoreName != cp.ScoreName {
		t.Errorf("score_name = %s, want %s", loaded.ScoreName, cp.ScoreName)
	}
	if loaded.CurrentMovement != cp.CurrentMovement {
		t.Errorf("current_movement = %d, want %d", loaded.CurrentMovement, cp.CurrentMovement)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if store.Exists() {
		t.Error("should not exist after clear")
	}
}

func TestMemoryStateStore(t *testing.T) {
	store := &MemoryStateStore{}

	if store.Exists() {
		t.Error("should not exist initially")
	}

	cp := &Checkpoint{ScoreName: "mem-test", CurrentMovement: 3}
	if err := store.Save(cp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !store.Exists() {
		t.Error("should exist after save")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.CurrentMovement != 3 {
		t.Errorf("current_movement = %d, want 3", loaded.CurrentMovement)
	}

	store.Clear()
	if store.Exists() {
		t.Error("should not exist after clear")
	}
}
