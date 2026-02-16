// Package agentsdk provides a lightweight Go client for AI agents to interact
// with the Legible Money chain. Designed for direct import by PicoClaw and other
// Go-based agent frameworks.
//
// Key features:
//   - Key management (generate, load, derive session keys)
//   - Transaction building for all agent-relevant tx types
//   - Query client for chain state
//   - Automatic gas estimation and fee calculation
//   - Heartbeat daemon for agent liveness
package agentsdk

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Protocol constants matching the chain.
const (
	ChainID         = "legible-devnet-1"
	DefaultNode     = "tcp://localhost:26657"
	DefaultGRPC     = "localhost:9090"
	Denom           = "ulgm"
	Bech32Prefix    = "lgm"
	BlockTime       = 2521 * time.Millisecond
	BlocksPerDay    = 34272
	MinClaimLen     = 20
	DefaultGasPrice = 1 // ulgm per gas unit
)

// AgentRole defines the verification role an agent plays.
type AgentRole string

const (
	RoleClaimer    AgentRole = "claimer"
	RoleVerifier   AgentRole = "verifier"
	RoleChallenger AgentRole = "challenger"
	RoleGuardian   AgentRole = "guardian"
)

// Verdict for claim verification.
type Verdict string

const (
	VerdictTrue      Verdict = "true"
	VerdictFalse     Verdict = "false"
	VerdictUncertain Verdict = "uncertain"
)

// AgentConfig holds configuration for an agent instance.
type AgentConfig struct {
	// NodeRPC is the CometBFT RPC endpoint.
	NodeRPC string `json:"node_rpc"`

	// GRPCAddr is the gRPC endpoint for queries.
	GRPCAddr string `json:"grpc_addr"`

	// ChainID is the chain identifier.
	ChainID string `json:"chain_id"`

	// KeyDir is where key material is stored.
	KeyDir string `json:"key_dir"`

	// GasPrice in ulgm per gas unit.
	GasPrice uint64 `json:"gas_price"`

	// AutoHeartbeat enables automatic heartbeat every HeartbeatInterval.
	AutoHeartbeat     bool          `json:"auto_heartbeat"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`

	// Domains this agent operates in.
	Domains []string `json:"domains"`

	// Role is the primary role of this agent.
	Role AgentRole `json:"role"`
}

// DefaultConfig returns a config suitable for devnet.
func DefaultConfig() AgentConfig {
	return AgentConfig{
		NodeRPC:           DefaultNode,
		GRPCAddr:          DefaultGRPC,
		ChainID:           ChainID,
		KeyDir:            filepath.Join(os.Getenv("HOME"), ".lgm-agent", "keys"),
		GasPrice:          DefaultGasPrice,
		AutoHeartbeat:     true,
		HeartbeatInterval: 1 * time.Hour,
		Domains:           []string{"general"},
		Role:              RoleVerifier,
	}
}

// LoadConfig loads config from a JSON file, falling back to defaults.
func LoadConfig(path string) (AgentConfig, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Agent represents an AI agent connected to the Legible Money chain.
type Agent struct {
	cfg AgentConfig

	// Identity
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
	address string // bech32 lgm1... address
	did     string // did:lgm:{hex}

	// Session key (scoped capabilities, auto-derived)
	sessionKey     ed25519.PrivateKey
	sessionAddress string

	// Salt store for commit-reveal verification
	salts   map[string]string // roundID -> hex salt
	saltsMu sync.Mutex

	// Heartbeat control
	heartbeatCancel context.CancelFunc

	// Query/Tx clients (interface-based for testability)
	queryClient QueryClient
	txClient    TxClient
}

// New creates a new Agent from config. Call Load() or GenerateKey() next.
func New(cfg AgentConfig) *Agent {
	return &Agent{
		cfg:   cfg,
		salts: make(map[string]string),
	}
}

// GenerateKey creates a new Ed25519 keypair and saves it to disk.
func (a *Agent) GenerateKey(name string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}
	a.privKey = priv
	a.pubKey = pub
	a.did = publicKeyToDID(pub)
	a.address = publicKeyToAddress(pub)

	// Persist
	if err := os.MkdirAll(a.cfg.KeyDir, 0700); err != nil {
		return fmt.Errorf("creating key dir: %w", err)
	}
	keyPath := filepath.Join(a.cfg.KeyDir, name+".key")
	return os.WriteFile(keyPath, priv.Seed(), 0600)
}

// LoadKey loads an existing key from disk.
func (a *Agent) LoadKey(name string) error {
	keyPath := filepath.Join(a.cfg.KeyDir, name+".key")
	seed, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("loading key %s: %w", name, err)
	}
	a.privKey = ed25519.NewKeyFromSeed(seed)
	a.pubKey = a.privKey.Public().(ed25519.PublicKey)
	a.did = publicKeyToDID(a.pubKey)
	a.address = publicKeyToAddress(a.pubKey)
	return nil
}

// Address returns the bech32 address.
func (a *Agent) Address() string { return a.address }

// DID returns the agent's decentralized identifier.
func (a *Agent) DID() string { return a.did }

// PubKeyHex returns the hex-encoded public key.
func (a *Agent) PubKeyHex() string { return hex.EncodeToString(a.pubKey) }

// --- Key Derivation ---

// publicKeyToDID derives did:lgm:{32-hex} from a public key.
func publicKeyToDID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "did:lgm:" + hex.EncodeToString(h[:16])
}

// publicKeyToAddress derives a bech32 lgm1... address from a public key.
// Uses the same derivation as Cosmos SDK: SHA256(pubkey)[:20] -> bech32.
func publicKeyToAddress(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	// Simplified bech32 encoding — in production, use cosmos sdk's bech32
	return fmt.Sprintf("lgm1%s", hex.EncodeToString(h[:20]))
}

// --- Salt Management for Commit-Reveal ---

// GenerateSalt creates and stores a cryptographic salt for a verification round.
func (a *Agent) GenerateSalt(roundID string) (string, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	hexSalt := hex.EncodeToString(salt)

	a.saltsMu.Lock()
	a.salts[roundID] = hexSalt
	a.saltsMu.Unlock()

	// Persist to disk
	saltDir := filepath.Join(a.cfg.KeyDir, "..", "salts")
	if err := os.MkdirAll(saltDir, 0700); err != nil {
		return hexSalt, nil // In-memory is fine
	}
	_ = os.WriteFile(filepath.Join(saltDir, roundID+".salt"), []byte(hexSalt), 0600)

	return hexSalt, nil
}

// GetSalt retrieves a stored salt for reveal phase.
func (a *Agent) GetSalt(roundID string) (string, bool) {
	a.saltsMu.Lock()
	defer a.saltsMu.Unlock()

	salt, ok := a.salts[roundID]
	if ok {
		return salt, true
	}

	// Try disk
	saltDir := filepath.Join(a.cfg.KeyDir, "..", "salts")
	data, err := os.ReadFile(filepath.Join(saltDir, roundID+".salt"))
	if err == nil {
		a.salts[roundID] = string(data)
		return string(data), true
	}

	return "", false
}

// ClearSalt removes a salt after successful reveal.
func (a *Agent) ClearSalt(roundID string) {
	a.saltsMu.Lock()
	delete(a.salts, roundID)
	a.saltsMu.Unlock()

	saltDir := filepath.Join(a.cfg.KeyDir, "..", "salts")
	_ = os.Remove(filepath.Join(saltDir, roundID+".salt"))
}

// ComputeCommitment computes SHA256(verdict || confidence || salt) for commit phase.
func ComputeCommitment(verdict Verdict, confidence uint32, salt string) string {
	data := fmt.Sprintf("%s%d%s", verdict, confidence, salt)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
