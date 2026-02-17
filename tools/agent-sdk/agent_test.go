package agentsdk

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.KeyDir = tmpDir

	agent := New(cfg)
	if err := agent.GenerateKey("test-agent"); err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Check key file exists
	keyPath := filepath.Join(tmpDir, "test-agent.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("Key file not created: %v", err)
	}

	// Check DID format
	if agent.DID() == "" {
		t.Fatal("DID is empty")
	}
	if agent.DID()[:8] != "did:lgm:" {
		t.Fatalf("DID has wrong prefix: %s", agent.DID())
	}

	// Check address format
	if agent.Address() == "" {
		t.Fatal("Address is empty")
	}
	if agent.Address()[:4] != "lgm1" {
		t.Fatalf("Address has wrong prefix: %s", agent.Address())
	}

	// Check pubkey hex
	if len(agent.PubKeyHex()) != 64 {
		t.Fatalf("PubKey hex wrong length: %d", len(agent.PubKeyHex()))
	}
}

func TestLoadKey(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.KeyDir = tmpDir

	// Generate first
	agent1 := New(cfg)
	if err := agent1.GenerateKey("reload-test"); err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Load in new agent
	agent2 := New(cfg)
	if err := agent2.LoadKey("reload-test"); err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	// Should have same identity
	if agent1.DID() != agent2.DID() {
		t.Fatalf("DIDs don't match: %s vs %s", agent1.DID(), agent2.DID())
	}
	if agent1.Address() != agent2.Address() {
		t.Fatalf("Addresses don't match: %s vs %s", agent1.Address(), agent2.Address())
	}
}

func TestSaltManagement(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.KeyDir = tmpDir

	agent := New(cfg)

	roundID := "round-123"

	// Generate salt
	salt, err := agent.GenerateSalt(roundID)
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if len(salt) != 64 { // 32 bytes hex
		t.Fatalf("Salt wrong length: %d", len(salt))
	}

	// Retrieve salt
	retrieved, ok := agent.GetSalt(roundID)
	if !ok {
		t.Fatal("GetSalt returned false")
	}
	if retrieved != salt {
		t.Fatalf("Salt mismatch: %s vs %s", retrieved, salt)
	}

	// Clear salt
	agent.ClearSalt(roundID)
	_, ok = agent.GetSalt(roundID)
	if ok {
		t.Fatal("Salt not cleared")
	}
}

func TestSaltPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.KeyDir = filepath.Join(tmpDir, "keys")

	agent1 := New(cfg)
	roundID := "round-persist"

	salt, err := agent1.GenerateSalt(roundID)
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	// New agent instance should find persisted salt
	agent2 := New(cfg)
	retrieved, ok := agent2.GetSalt(roundID)
	if !ok {
		t.Fatal("Persisted salt not found")
	}
	if retrieved != salt {
		t.Fatalf("Persisted salt mismatch: %s vs %s", retrieved, salt)
	}
}

func TestComputeCommitment(t *testing.T) {
	roundID := "test-round-1"
	// Deterministic: same inputs should produce same hash
	c1 := ComputeCommitment(roundID, VerdictTrue, 800000, "abc123")
	c2 := ComputeCommitment(roundID, VerdictTrue, 800000, "abc123")
	if c1 != c2 {
		t.Fatal("Commitment not deterministic")
	}

	// Different inputs should produce different hashes
	c3 := ComputeCommitment(roundID, VerdictFalse, 800000, "abc123")
	if c1 == c3 {
		t.Fatal("Different verdicts produced same commitment")
	}

	c4 := ComputeCommitment(roundID, VerdictTrue, 800000, "def456")
	if c1 == c4 {
		t.Fatal("Different salts produced same commitment")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ChainID != "legible-devnet-1" {
		t.Fatalf("Wrong chain ID: %s", cfg.ChainID)
	}
	if cfg.GasPrice != 1 {
		t.Fatalf("Wrong gas price: %d", cfg.GasPrice)
	}
	if cfg.Role != RoleVerifier {
		t.Fatalf("Wrong default role: %s", cfg.Role)
	}
}

func TestMinClaimLengthValidation(t *testing.T) {
	agent := New(DefaultConfig())
	// Use a mock tx client
	mock := &mockTxClient{}
	agent.SetTxClient(mock)

	// Short claim should fail
	_, err := agent.SubmitClaim(context.Background(), "too short", "general", "computational", 1000000)
	if err == nil {
		t.Fatal("Expected error for short claim")
	}
}

// mockTxClient for testing
type mockTxClient struct{}

func (m *mockTxClient) RegisterAccount(ctx context.Context, pubKeyHex string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) Heartbeat(ctx context.Context) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) SubmitClaim(ctx context.Context, req SubmitClaimRequest) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) SubmitCommitment(ctx context.Context, roundID, commitment string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) SubmitReveal(ctx context.Context, roundID string, verdict Verdict, confidence uint32, salt string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) ChallengeFact(ctx context.Context, factID, reason string, stake uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) CreateHome(ctx context.Context, name string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) UpdateMemoryCID(ctx context.Context, homeID, cid string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) Stake(ctx context.Context, validatorAddr string, amount uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) DeployContract(ctx context.Context, bytecodeHex string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) CallContract(ctx context.Context, contractAddr, inputDataHex string, gasLimit uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) ScheduleExecution(ctx context.Context, contractAddr, method, payload string, atBlock, maxGas uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) ConfigureGuardian(ctx context.Context, homeID, strategy string, autoDefend bool, deadmanThreshold uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) RegisterHomeKey(ctx context.Context, homeID, keyHash, role string, permissions []string, expiresAt uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) StartSession(ctx context.Context, homeID, keyHash string, permissions []string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) EndSession(ctx context.Context, homeID, sessionID string) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) SetSpendingLimit(ctx context.Context, homeID, keyType string, maxAmount, periodBlocks uint64) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) SubmitLIP(ctx context.Context, req SubmitLIPRequest) (string, error) {
	return "mock-tx-hash", nil
}
func (m *mockTxClient) CreateVesting(ctx context.Context, req CreateVestingRequest) (string, error) {
	return "mock-tx-hash", nil
}
