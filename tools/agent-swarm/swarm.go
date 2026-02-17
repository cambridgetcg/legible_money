// Package swarm provides multi-agent coordination for PicoClaw agents
// participating in the Legible Money protocol.
//
// A swarm is a group of agents that coordinate to maximize collective
// rewards through role specialization and task distribution.
//
// Roles:
//   - Claimer:    Submits knowledge claims (high stake, high reward)
//   - Verifier:   Evaluates claims in PoT rounds (steady income)
//   - Challenger: Challenges weak facts (high risk, high reward)
//   - Guardian:   Tier 3 validators with governance + emergency powers
//
// Economics:
//   - Bootstrap phase: gas-free for first 480,000 blocks (~14 days)
//   - Verifier agents earn ~1 LGM per correct verification
//   - Heartbeat cost: ~0.36 LGM/day per agent
//   - Minimum viable balance: 100 LGM (discovery stake) + operating reserve
package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// SwarmConfig defines the configuration for a multi-agent swarm.
type SwarmConfig struct {
	// Name is the swarm identifier.
	Name string `json:"name"`

	// BaseDir is the root directory for all agent workspaces.
	BaseDir string `json:"base_dir"`

	// NodeRPC is the chain RPC endpoint (shared by all agents).
	NodeRPC string `json:"node_rpc"`

	// ChainID is the chain identifier.
	ChainID string `json:"chain_id"`

	// Agents defines the agent roster.
	Agents []AgentSpec `json:"agents"`

	// EconomicParams controls swarm economics.
	Economics EconomicParams `json:"economics"`
}

// AgentSpec defines a single agent in the swarm.
type AgentSpec struct {
	// Name is the agent's display name.
	Name string `json:"name"`

	// KeyName is the keyring key name.
	KeyName string `json:"key_name"`

	// Role is the agent's primary role.
	Role agentsdk.AgentRole `json:"role"`

	// Domains the agent operates in.
	Domains []string `json:"domains"`

	// InitialStake is the ulgm to stake on registration.
	InitialStake uint64 `json:"initial_stake"`

	// Enabled controls whether this agent is active.
	Enabled bool `json:"enabled"`
}

// EconomicParams controls swarm economic behavior.
type EconomicParams struct {
	// MinOperatingBalance is the minimum ulgm before alerting.
	MinOperatingBalance uint64 `json:"min_operating_balance"`

	// MaxDailyGasSpend is the daily gas budget in ulgm.
	MaxDailyGasSpend uint64 `json:"max_daily_gas_spend"`

	// StakeReserveRatio is the fraction of balance to keep liquid (0-100).
	StakeReserveRatio uint32 `json:"stake_reserve_ratio"`

	// RebalanceInterval is how often to check economic health.
	RebalanceInterval time.Duration `json:"rebalance_interval"`
}

// DefaultSwarmConfig returns a minimal swarm config for devnet testing.
func DefaultSwarmConfig(baseDir string) SwarmConfig {
	return SwarmConfig{
		Name:    "lgm-devnet-swarm",
		BaseDir: baseDir,
		NodeRPC: agentsdk.DefaultNode,
		ChainID: agentsdk.ChainID,
		Agents: []AgentSpec{
			{
				Name:         "claimer-alpha",
				KeyName:      "claimer-alpha",
				Role:         agentsdk.RoleClaimer,
				Domains:      []string{"mathematics", "computer_science"},
				InitialStake: 100_000_000, // 100 LGM
				Enabled:      true,
			},
			{
				Name:         "verifier-one",
				KeyName:      "verifier-one",
				Role:         agentsdk.RoleVerifier,
				Domains:      []string{"mathematics", "computer_science", "physics"},
				InitialStake: 100_000_000, // 100 LGM
				Enabled:      true,
			},
			{
				Name:         "verifier-two",
				KeyName:      "verifier-two",
				Role:         agentsdk.RoleVerifier,
				Domains:      []string{"mathematics", "physics", "general"},
				InitialStake: 100_000_000, // 100 LGM
				Enabled:      true,
			},
			{
				Name:         "challenger-beta",
				KeyName:      "challenger-beta",
				Role:         agentsdk.RoleChallenger,
				Domains:      []string{"general"},
				InitialStake: 100_000_000, // 100 LGM
				Enabled:      false,       // Enable after bootstrap
			},
		},
		Economics: EconomicParams{
			MinOperatingBalance: 10_000_000, // 10 LGM
			MaxDailyGasSpend:    5_000_000,  // 5 LGM
			StakeReserveRatio:   20,         // Keep 20% liquid
			RebalanceInterval:   30 * time.Minute,
		},
	}
}

// Swarm manages a group of coordinated agents.
type Swarm struct {
	cfg    SwarmConfig
	agents map[string]*ManagedAgent
	mu     sync.RWMutex

	// Task queue for cross-agent coordination
	taskQueue chan SwarmTask
}

// ManagedAgent wraps an agent with swarm management state.
type ManagedAgent struct {
	Spec    AgentSpec
	Agent   *agentsdk.Agent
	Status  AgentStatus
	Metrics AgentMetrics
}

// AgentStatus tracks the runtime status of a managed agent.
type AgentStatus string

const (
	StatusPending    AgentStatus = "pending"    // Not yet registered
	StatusActive     AgentStatus = "active"     // Running and healthy
	StatusDegraded   AgentStatus = "degraded"   // Running but low balance
	StatusPaused     AgentStatus = "paused"     // Temporarily paused
	StatusFailed     AgentStatus = "failed"     // Error state
)

// AgentMetrics tracks operational metrics for a managed agent.
type AgentMetrics struct {
	ClaimsSubmitted    int    `json:"claims_submitted"`
	VerificationsCount int    `json:"verifications_count"`
	ChallengesIssued   int    `json:"challenges_issued"`
	RewardsEarned      uint64 `json:"rewards_earned_ulgm"`
	GasSpent           uint64 `json:"gas_spent_ulgm"`
	LastActivity       string `json:"last_activity"`
}

// SwarmTask represents a task distributed across agents.
type SwarmTask struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // "verify", "claim", "challenge"
	Payload  map[string]string `json:"payload"`
	Priority int               `json:"priority"` // Higher = more urgent
	AssignTo agentsdk.AgentRole `json:"assign_to"`
}

// NewSwarm creates a new swarm from config.
func NewSwarm(cfg SwarmConfig) *Swarm {
	return &Swarm{
		cfg:       cfg,
		agents:    make(map[string]*ManagedAgent),
		taskQueue: make(chan SwarmTask, 100),
	}
}

// Init initializes all agents in the swarm.
func (s *Swarm) Init() error {
	for _, spec := range s.cfg.Agents {
		if !spec.Enabled {
			continue
		}

		agentCfg := agentsdk.AgentConfig{
			NodeRPC:           s.cfg.NodeRPC,
			GRPCAddr:          agentsdk.DefaultGRPC,
			ChainID:           s.cfg.ChainID,
			KeyDir:            filepath.Join(s.cfg.BaseDir, spec.KeyName, "keys"),
			GasPrice:          agentsdk.DefaultGasPrice,
			AutoHeartbeat:     true,
			HeartbeatInterval: 1 * time.Hour,
			Domains:           spec.Domains,
			Role:              spec.Role,
		}

		agent := agentsdk.New(agentCfg)

		// Try to load existing key, generate if not found
		if err := agent.LoadKey(spec.KeyName); err != nil {
			if err := agent.GenerateKey(spec.KeyName); err != nil {
				return fmt.Errorf("init agent %s: %w", spec.Name, err)
			}
		}

		// Set up CLI client
		client := agentsdk.NewCLIClient(spec.KeyName)
		client.Node = s.cfg.NodeRPC
		client.ChainID = s.cfg.ChainID
		agent.SetQueryClient(client)
		agent.SetTxClient(client)

		s.mu.Lock()
		s.agents[spec.Name] = &ManagedAgent{
			Spec:   spec,
			Agent:  agent,
			Status: StatusPending,
		}
		s.mu.Unlock()
	}
	return nil
}

// RegisterAll registers all pending agents on-chain.
func (s *Swarm) RegisterAll(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, ma := range s.agents {
		if ma.Status != StatusPending {
			continue
		}

		err := ma.Agent.Register(ctx, ma.Spec.Name, ma.Spec.Domains, ma.Spec.InitialStake)
		if err != nil {
			ma.Status = StatusFailed
			return fmt.Errorf("registering %s: %w", name, err)
		}
		ma.Status = StatusActive
	}
	return nil
}

// AddAgent adds a pre-configured managed agent to the swarm.
func (s *Swarm) AddAgent(ma *ManagedAgent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[ma.Spec.Name] = ma
}

// GetAgent returns a managed agent by name.
func (s *Swarm) GetAgent(name string) (*ManagedAgent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ma, ok := s.agents[name]
	return ma, ok
}

// ListAgents returns all managed agents.
func (s *Swarm) ListAgents() []*ManagedAgent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ManagedAgent, 0, len(s.agents))
	for _, ma := range s.agents {
		result = append(result, ma)
	}
	return result
}

// SubmitTask adds a task to the swarm queue.
func (s *Swarm) SubmitTask(task SwarmTask) {
	s.taskQueue <- task
}

// AgentsByRole returns all agents with a specific role.
func (s *Swarm) AgentsByRole(role agentsdk.AgentRole) []*ManagedAgent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ManagedAgent
	for _, ma := range s.agents {
		if ma.Spec.Role == role && ma.Status == StatusActive {
			result = append(result, ma)
		}
	}
	return result
}

// SaveState persists swarm state to disk for crash recovery.
func (s *Swarm) SaveState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := struct {
		Config SwarmConfig                `json:"config"`
		Agents map[string]AgentMetrics    `json:"agent_metrics"`
		Status map[string]AgentStatus     `json:"agent_status"`
	}{
		Config: s.cfg,
		Agents: make(map[string]AgentMetrics),
		Status: make(map[string]AgentStatus),
	}

	for name, ma := range s.agents {
		state.Agents[name] = ma.Metrics
		state.Status[name] = ma.Status
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	statePath := filepath.Join(s.cfg.BaseDir, "swarm-state.json")
	return os.WriteFile(statePath, data, 0644)
}

// LoadState restores swarm state from disk.
func (s *Swarm) LoadState() error {
	statePath := filepath.Join(s.cfg.BaseDir, "swarm-state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state to restore
		}
		return fmt.Errorf("reading state: %w", err)
	}

	var state struct {
		Agents map[string]AgentMetrics `json:"agent_metrics"`
		Status map[string]AgentStatus  `json:"agent_status"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parsing state: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for name, metrics := range state.Agents {
		if ma, ok := s.agents[name]; ok {
			ma.Metrics = metrics
		}
	}
	for name, status := range state.Status {
		if ma, ok := s.agents[name]; ok {
			ma.Status = status
		}
	}

	return nil
}
