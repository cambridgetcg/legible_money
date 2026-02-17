package orchestrator

import (
	"os"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

// testScore builds a minimal valid score for testing.
func testScore(t *testing.T) *Score {
	t.Helper()
	return &Score{
		Name: "test-score",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 2, MaxCount: 5, Domains: []string{"general"}},
				{Name: SectionBrass, MinCount: 1, MaxCount: 3, Domains: []string{"general"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "phase-1",
				Parts: []PartSpec{
					{ID: "register-v", Section: SectionStrings, Action: ActionRegister, RepeatPerAgent: true},
					{ID: "register-c", Section: SectionBrass, Action: ActionRegister, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
			{
				Name: "phase-2",
				Parts: []PartSpec{
					{ID: "stake-v", Section: SectionStrings, Action: ActionStake, Params: map[string]string{"validator_addr": "lgmval1test", "amount": "1000"}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       50 * time.Millisecond,
			StaggerDelay:       0,
			CheckpointInterval: time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 1000,
			MaxTotalGasSpend:    100000,
			StakePerAgent:       1000,
		},
	}
}

// testSwarm creates a swarm initialized with rehearsal clients.
// Returns the swarm, cleanup function, and rehearsal clients.
func testSwarm(t *testing.T, score *Score) (*swarm.Swarm, func(), *RehearsalQueryClient, *RehearsalTxClient) {
	t.Helper()
	qc := NewRehearsalQueryClient()
	tc := NewRehearsalTxClient()

	sw, tmpDir, err := BuildRehearsalSwarm(score, qc, tc)
	if err != nil {
		t.Fatalf("build rehearsal swarm: %v", err)
	}

	cleanup := func() { os.RemoveAll(tmpDir) }
	return sw, cleanup, qc, tc
}

// testConductor builds a conductor with rehearsal executor and memory state store.
func testConductor(t *testing.T) (*Conductor, func(), *RehearsalQueryClient) {
	t.Helper()
	score := testScore(t)
	sw, cleanup, qc, _ := testSwarm(t, score)

	var events []Event
	handler := func(e Event) {
		events = append(events, e)
	}
	_ = events // available for inspection in tests

	conductor, err := NewConductor(score, sw,
		WithEventHandler(handler),
		WithExecutor(&RehearsalExecutor{}),
		WithStateStore(&MemoryStateStore{}),
	)
	if err != nil {
		cleanup()
		t.Fatalf("new conductor: %v", err)
	}

	return conductor, cleanup, qc
}

// testManagedAgents builds standalone ManagedAgents for section tests.
func testManagedAgents(t *testing.T, count int, role agentsdk.AgentRole) ([]*swarm.ManagedAgent, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "orchestrator-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	qc := NewRehearsalQueryClient()
	tc := NewRehearsalTxClient()

	agents := make([]*swarm.ManagedAgent, count)
	for i := 0; i < count; i++ {
		name := agentName(role, i)
		cfg := agentsdk.AgentConfig{
			KeyDir:  tmpDir,
			Role:    role,
			Domains: []string{"general"},
		}
		agent := agentsdk.New(cfg)
		if err := agent.GenerateKey(name); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("generate key %s: %v", name, err)
		}
		agent.SetQueryClient(qc)
		agent.SetTxClient(tc)

		agents[i] = &swarm.ManagedAgent{
			Spec: swarm.AgentSpec{
				Name:    name,
				KeyName: name,
				Role:    role,
				Domains: []string{"general"},
				Enabled: true,
			},
			Agent:  agent,
			Status: swarm.StatusActive,
		}
	}

	cleanup := func() { os.RemoveAll(tmpDir) }
	return agents, cleanup
}

func agentName(role agentsdk.AgentRole, idx int) string {
	return string(role) + "-" + string(rune('A'+idx))
}
