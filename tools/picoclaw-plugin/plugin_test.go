package picoclaw

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

func TestPluginToolCount(t *testing.T) {
	cfg := agentsdk.DefaultConfig()
	agent := agentsdk.New(cfg)
	plugin := NewPlugin(agent, cfg)

	tools := plugin.Tools()
	if len(tools) != 10 {
		t.Fatalf("Expected 10 tools, got %d", len(tools))
	}

	// Verify all tool names are unique
	names := make(map[string]bool)
	for _, tool := range tools {
		if names[tool.Name()] {
			t.Fatalf("Duplicate tool name: %s", tool.Name())
		}
		names[tool.Name()] = true
	}
}

func TestToolNames(t *testing.T) {
	cfg := agentsdk.DefaultConfig()
	agent := agentsdk.New(cfg)
	plugin := NewPlugin(agent, cfg)

	expected := []string{
		"chain_claim",
		"chain_verify",
		"chain_query_fact",
		"chain_challenge",
		"chain_heartbeat",
		"chain_balance",
		"chain_stake",
		"chain_vote",
		"chain_home_sync",
		"chain_discover",
	}

	tools := plugin.Tools()
	for i, tool := range tools {
		if tool.Name() != expected[i] {
			t.Errorf("Tool %d: expected %s, got %s", i, expected[i], tool.Name())
		}
	}
}

func TestToolDescriptions(t *testing.T) {
	cfg := agentsdk.DefaultConfig()
	agent := agentsdk.New(cfg)
	plugin := NewPlugin(agent, cfg)

	for _, tool := range plugin.Tools() {
		desc := tool.Description()
		if desc == "" {
			t.Errorf("Tool %s has empty description", tool.Name())
		}
		if len(desc) < 20 {
			t.Errorf("Tool %s description too short: %s", tool.Name(), desc)
		}
	}
}

func TestInitWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "agent-workspace")

	err := InitWorkspace(wsDir, "test-agent", "testkey", []string{"mathematics", "physics"})
	if err != nil {
		t.Fatalf("InitWorkspace failed: %v", err)
	}

	// Verify directory structure
	dirs := []string{
		"skills",
		"memory",
		"memory/facts",
		"memory/verifications",
		"logs",
	}
	for _, d := range dirs {
		path := filepath.Join(wsDir, d)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Directory not created: %s", d)
		}
	}

	// Verify config
	cfg, err := LoadWorkspaceConfig(wsDir)
	if err != nil {
		t.Fatalf("LoadWorkspaceConfig failed: %v", err)
	}
	if cfg.AgentName != "test-agent" {
		t.Errorf("Wrong agent name: %s", cfg.AgentName)
	}
	if cfg.KeyName != "testkey" {
		t.Errorf("Wrong key name: %s", cfg.KeyName)
	}
	if len(cfg.Domains) != 2 {
		t.Errorf("Wrong domain count: %d", len(cfg.Domains))
	}

	// Verify HEARTBEAT.md exists
	if _, err := os.Stat(filepath.Join(wsDir, "HEARTBEAT.md")); err != nil {
		t.Error("HEARTBEAT.md not created")
	}

	// Verify env script
	if _, err := os.Stat(filepath.Join(wsDir, "skills", "lgm-env.sh")); err != nil {
		t.Error("lgm-env.sh not created")
	}
}

func TestComputeMemoryCID(t *testing.T) {
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memDir, 0755)

	// Write some test data
	os.WriteFile(filepath.Join(memDir, "test.txt"), []byte("hello world"), 0644)

	cid, err := ComputeMemoryCID(memDir)
	if err != nil {
		t.Fatalf("ComputeMemoryCID failed: %v", err)
	}
	if len(cid) < 10 {
		t.Fatalf("CID too short: %s", cid)
	}
	if cid[:4] != "bafk" {
		t.Fatalf("CID wrong prefix: %s", cid)
	}

	// Deterministic: same content -> same CID
	cid2, _ := ComputeMemoryCID(memDir)
	if cid != cid2 {
		t.Fatal("CID not deterministic")
	}

	// Different content -> different CID
	os.WriteFile(filepath.Join(memDir, "test2.txt"), []byte("different"), 0644)
	cid3, _ := ComputeMemoryCID(memDir)
	if cid == cid3 {
		t.Fatal("Different content produced same CID")
	}
}

func TestSimpleEvaluator(t *testing.T) {
	eval := &SimpleEvaluator{
		DefaultVerdict:    agentsdk.VerdictTrue,
		DefaultConfidence: 800000,
	}

	verdict, confidence, err := eval.Evaluate(context.Background(), "claim-123")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Errorf("Wrong verdict: %s", verdict)
	}
	if confidence != 800000 {
		t.Errorf("Wrong confidence: %d", confidence)
	}
}

func TestDomainEvaluator(t *testing.T) {
	eval := &DomainEvaluator{
		EvalFunc: func(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
			return agentsdk.VerdictFalse, 900000, nil
		},
	}

	verdict, confidence, err := eval.Evaluate(context.Background(), "claim-456")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if verdict != agentsdk.VerdictFalse {
		t.Errorf("Wrong verdict: %s", verdict)
	}
	if confidence != 900000 {
		t.Errorf("Wrong confidence: %d", confidence)
	}
}

func TestDomainEvaluatorNilFunc(t *testing.T) {
	eval := &DomainEvaluator{}

	verdict, confidence, err := eval.Evaluate(context.Background(), "claim-789")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if verdict != agentsdk.VerdictUncertain {
		t.Errorf("Wrong default verdict: %s", verdict)
	}
	if confidence != 500000 {
		t.Errorf("Wrong default confidence: %d", confidence)
	}
}

func TestParseUint(t *testing.T) {
	tests := []struct {
		input    string
		fallback uint64
		expected uint64
	}{
		{"", 100, 100},
		{"0", 100, 100},
		{"42", 100, 42},
		{"1000000", 0, 1000000},
		{"invalid", 50, 50},
	}

	for _, tt := range tests {
		got := parseUint(tt.input, tt.fallback)
		if got != tt.expected {
			t.Errorf("parseUint(%q, %d) = %d, want %d", tt.input, tt.fallback, got, tt.expected)
		}
	}
}
