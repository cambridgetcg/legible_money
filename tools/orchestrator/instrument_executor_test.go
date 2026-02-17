package orchestrator

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

func TestInstrumentExecutorPrepareBeforeExecute(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleVerifier)
	defer cleanup()
	ma := agents[0]

	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-1",
		Category: "formal",
		Domain:   "general",
	})

	stringInst := NewStringInstrument(StrategyConservative, nil, resolver)
	instruments := map[SectionName]Instrument{
		SectionStrings: stringInst,
	}

	exec := NewInstrumentExecutor(&RehearsalExecutor{}, instruments)

	ctx := context.Background()
	result, err := exec.Execute(ctx, ma.Agent, ActionVerify, map[string]string{
		"round_id": "round-1",
		"claim_id": "claim-1",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// StringInstrument should have evaluated and cached
	if stringInst.CacheSize() != 1 {
		t.Fatalf("expected 1 cached evaluation, got %d", stringInst.CacheSize())
	}
}

func TestInstrumentExecutorFallthrough(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleVerifier)
	defer cleanup()
	ma := agents[0]

	// No instruments registered
	exec := NewInstrumentExecutor(&RehearsalExecutor{}, nil)

	ctx := context.Background()
	result, err := exec.Execute(ctx, ma.Agent, ActionHeartbeat, nil)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success for heartbeat fallthrough")
	}
}

func TestInstrumentExecutorError(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleChallenger)
	defer cleanup()
	ma := agents[0]

	// Percussion instrument with no scanner — will fail to find candidates
	inst := NewPercussionInstrument(
		NewDefaultChallengeDetector(100_000),
		NewMockFactScanner(), // Empty scanner
		[]string{"general"},
	)
	instruments := map[SectionName]Instrument{
		SectionPercussion: inst,
	}

	exec := NewInstrumentExecutor(&RehearsalExecutor{}, instruments)

	ctx := context.Background()
	result, err := exec.Execute(ctx, ma.Agent, ActionChallenge, map[string]string{})

	if err == nil {
		t.Fatal("expected error when no candidates found")
	}
	if result == nil {
		t.Fatal("expected non-nil result even on error")
	}
	if result.Success {
		t.Fatal("expected failure result")
	}
}

func TestInstrumentExecutorBrass(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleClaimer)
	defer cleanup()
	ma := agents[0]

	strategy := NewTemplateClaimStrategy(map[string][]ClaimTemplate{
		"general": {
			{Text: "A sufficiently long test claim for brass instrument testing", Category: "computational", Stake: 5_000_000},
		},
	})
	brassInst := NewBrassInstrument(strategy, []string{"general"}, 1_000_000)

	instruments := map[SectionName]Instrument{
		SectionBrass: brassInst,
	}

	exec := NewInstrumentExecutor(&RehearsalExecutor{}, instruments)

	ctx := context.Background()
	result, err := exec.Execute(ctx, ma.Agent, ActionSubmitClaim, map[string]string{
		"domain": "general",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestInstrumentExecutorWind(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleGuardian)
	defer cleanup()
	ma := agents[0]

	windInst := NewWindInstrument(DefenseActive, 10_000)

	instruments := map[SectionName]Instrument{
		SectionWinds: windInst,
	}

	exec := NewInstrumentExecutor(&RehearsalExecutor{}, instruments)

	ctx := context.Background()
	result, err := exec.Execute(ctx, ma.Agent, ActionCreateHome, map[string]string{
		"name": "test-home",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}
