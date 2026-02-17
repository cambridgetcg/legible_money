package orchestrator

import "testing"

func TestRepoSymphonyScoreValidates(t *testing.T) {
	score := RepoSymphonyScore(3, 2, 1, 1)

	if err := score.Validate(); err != nil {
		t.Fatalf("RepoSymphonyScore: %v", err)
	}

	if score.Name != "repo-symphony" {
		t.Errorf("name = %s, want repo-symphony", score.Name)
	}
	if len(score.Ensemble.Sections) != 4 {
		t.Errorf("sections = %d, want 4", len(score.Ensemble.Sections))
	}
	if len(score.Movements) != 5 {
		t.Errorf("movements = %d, want 5", len(score.Movements))
	}

	// Check each movement has proper IDs.
	for _, m := range score.Movements {
		for _, p := range m.Parts {
			if p.ID == "" {
				t.Errorf("movement %q has part with empty ID", m.Name)
			}
		}
	}
}

func TestVerificationConcertoScoreValidates(t *testing.T) {
	score := VerificationConcertoScore(5)

	if err := score.Validate(); err != nil {
		t.Fatalf("VerificationConcertoScore: %v", err)
	}

	if score.Name != "verification-concerto" {
		t.Errorf("name = %s, want verification-concerto", score.Name)
	}
	if len(score.Ensemble.Sections) != 1 {
		t.Errorf("sections = %d, want 1 (strings only)", len(score.Ensemble.Sections))
	}
	if score.Ensemble.Sections[0].Name != SectionStrings {
		t.Errorf("section = %s, want strings", score.Ensemble.Sections[0].Name)
	}
	if len(score.Movements) != 4 {
		t.Errorf("movements = %d, want 4", len(score.Movements))
	}
}

func TestChallengeStudyScoreValidates(t *testing.T) {
	score := ChallengeStudyScore(2)

	if err := score.Validate(); err != nil {
		t.Fatalf("ChallengeStudyScore: %v", err)
	}

	if score.Name != "challenge-study" {
		t.Errorf("name = %s, want challenge-study", score.Name)
	}
	if len(score.Ensemble.Sections) != 1 {
		t.Errorf("sections = %d, want 1 (percussion only)", len(score.Ensemble.Sections))
	}
	if score.Ensemble.Sections[0].Name != SectionPercussion {
		t.Errorf("section = %s, want percussion", score.Ensemble.Sections[0].Name)
	}
	if len(score.Movements) != 4 {
		t.Errorf("movements = %d, want 4", len(score.Movements))
	}

	// Verify challenge actions are present.
	hasChallengeAction := false
	for _, m := range score.Movements {
		for _, p := range m.Parts {
			if p.Action == ActionChallenge {
				hasChallengeAction = true
			}
		}
	}
	if !hasChallengeAction {
		t.Error("expected at least one challenge action in challenge study score")
	}
}

func TestRepoSymphonyScoreRehearsal(t *testing.T) {
	score := RepoSymphonyScore(2, 1, 1, 1)

	events, err := Rehearse(score)
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events from rehearsal")
	}

	var hasComplete bool
	movementsDone := 0
	for _, e := range events {
		if e.Type == EventPerformCompleted {
			hasComplete = true
		}
		if e.Type == EventMovementCompleted {
			movementsDone++
		}
	}

	if !hasComplete {
		t.Error("missing perform_completed event")
	}
	if movementsDone != 5 {
		t.Errorf("completed %d movements, want 5", movementsDone)
	}
}
