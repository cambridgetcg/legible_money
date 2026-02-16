package swarm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

func TestDefaultSwarmConfig(t *testing.T) {
	cfg := DefaultSwarmConfig("/tmp/test-swarm")

	if cfg.Name != "lgm-devnet-swarm" {
		t.Errorf("Wrong swarm name: %s", cfg.Name)
	}
	if len(cfg.Agents) != 4 {
		t.Errorf("Wrong agent count: %d", len(cfg.Agents))
	}

	// Check roles
	roles := make(map[agentsdk.AgentRole]int)
	for _, a := range cfg.Agents {
		roles[a.Role]++
	}
	if roles[agentsdk.RoleClaimer] != 1 {
		t.Error("Expected 1 claimer")
	}
	if roles[agentsdk.RoleVerifier] != 2 {
		t.Error("Expected 2 verifiers")
	}
	if roles[agentsdk.RoleChallenger] != 1 {
		t.Error("Expected 1 challenger")
	}

	// Check enabled count
	enabled := 0
	for _, a := range cfg.Agents {
		if a.Enabled {
			enabled++
		}
	}
	if enabled != 3 {
		t.Errorf("Expected 3 enabled agents, got %d", enabled)
	}
}

func TestSwarmInit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	swarm := NewSwarm(cfg)
	if err := swarm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	agents := swarm.ListAgents()
	// Only enabled agents should be initialized (3 of 4)
	if len(agents) != 3 {
		t.Fatalf("Expected 3 initialized agents, got %d", len(agents))
	}

	// All should be pending
	for _, ma := range agents {
		if ma.Status != StatusPending {
			t.Errorf("Agent %s should be pending, got %s", ma.Spec.Name, ma.Status)
		}
	}
}

func TestSwarmGetAgent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	swarm := NewSwarm(cfg)
	swarm.Init()

	ma, ok := swarm.GetAgent("verifier-one")
	if !ok {
		t.Fatal("verifier-one not found")
	}
	if ma.Spec.Role != agentsdk.RoleVerifier {
		t.Errorf("Wrong role: %s", ma.Spec.Role)
	}

	_, ok = swarm.GetAgent("nonexistent")
	if ok {
		t.Fatal("nonexistent agent should not be found")
	}
}

func TestAgentsByRole(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	swarm := NewSwarm(cfg)
	swarm.Init()

	// Set agents to active for role query
	for _, ma := range swarm.ListAgents() {
		ma.Status = StatusActive
	}

	verifiers := swarm.AgentsByRole(agentsdk.RoleVerifier)
	if len(verifiers) != 2 {
		t.Errorf("Expected 2 verifiers, got %d", len(verifiers))
	}

	claimers := swarm.AgentsByRole(agentsdk.RoleClaimer)
	if len(claimers) != 1 {
		t.Errorf("Expected 1 claimer, got %d", len(claimers))
	}

	// Challenger is disabled, so 0 even if we look
	challengers := swarm.AgentsByRole(agentsdk.RoleChallenger)
	if len(challengers) != 0 {
		t.Errorf("Expected 0 challengers (disabled), got %d", len(challengers))
	}
}

func TestSwarmSaveLoadState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	// Create and init swarm
	s1 := NewSwarm(cfg)
	s1.Init()

	// Set some metrics
	for _, ma := range s1.ListAgents() {
		ma.Status = StatusActive
		ma.Metrics.ClaimsSubmitted = 5
		ma.Metrics.VerificationsCount = 10
		ma.Metrics.RewardsEarned = 50_000_000
	}

	// Save
	if err := s1.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, "swarm-state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatal("State file not created")
	}

	// Create new swarm and load state
	s2 := NewSwarm(cfg)
	s2.Init()
	if err := s2.LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify metrics restored
	ma, ok := s2.GetAgent("verifier-one")
	if !ok {
		t.Fatal("verifier-one not found after load")
	}
	if ma.Metrics.VerificationsCount != 10 {
		t.Errorf("Metrics not restored: verifications=%d", ma.Metrics.VerificationsCount)
	}
	if ma.Metrics.RewardsEarned != 50_000_000 {
		t.Errorf("Metrics not restored: rewards=%d", ma.Metrics.RewardsEarned)
	}
}

func TestHealthCheck(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	swarm := NewSwarm(cfg)
	swarm.Init()

	// Set metrics for health check
	for _, ma := range swarm.ListAgents() {
		ma.Status = StatusActive
		ma.Metrics.RewardsEarned = 10_000_000
		ma.Metrics.GasSpent = 500_000
	}

	report := swarm.HealthCheck(context.Background())

	if report.TotalAgents != 3 {
		t.Errorf("Wrong total agents: %d", report.TotalAgents)
	}
	if report.ActiveAgents != 3 {
		t.Errorf("Wrong active agents: %d", report.ActiveAgents)
	}
	if report.TotalRewards != 30_000_000 {
		t.Errorf("Wrong total rewards: %d", report.TotalRewards)
	}
	if report.TotalGasSpent != 1_500_000 {
		t.Errorf("Wrong total gas: %d", report.TotalGasSpent)
	}
	if !report.Sustainable {
		t.Error("Should be sustainable (rewards > gas)")
	}
	if report.NetPosition != 28_500_000 {
		t.Errorf("Wrong net position: %d", report.NetPosition)
	}
}

func TestDefaultBootstrapStrategy(t *testing.T) {
	strategy := DefaultBootstrapStrategy()

	if !strategy.RegistrationPhase.RegisterAll {
		t.Error("Should register all by default")
	}
	if strategy.StakingPhase.StakePerAgent != 100_000_000 {
		t.Errorf("Wrong stake per agent: %d", strategy.StakingPhase.StakePerAgent)
	}
	if strategy.VerificationPhase.AccuracyThreshold != 770000 {
		t.Errorf("Wrong accuracy threshold: %d", strategy.VerificationPhase.AccuracyThreshold)
	}
	if strategy.ExpansionPhase.MaxAgents != 20 {
		t.Errorf("Wrong max agents: %d", strategy.ExpansionPhase.MaxAgents)
	}
}

func TestSubmitTask(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultSwarmConfig(tmpDir)

	swarm := NewSwarm(cfg)

	task := SwarmTask{
		ID:       "task-1",
		Type:     "verify",
		Payload:  map[string]string{"round_id": "round-123"},
		Priority: 1,
		AssignTo: agentsdk.RoleVerifier,
	}

	// Should not block (buffered channel)
	swarm.SubmitTask(task)

	// Read it back
	received := <-swarm.taskQueue
	if received.ID != "task-1" {
		t.Errorf("Wrong task ID: %s", received.ID)
	}
}
