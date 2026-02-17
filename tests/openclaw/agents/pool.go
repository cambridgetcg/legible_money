//go:build openclaw

package agents

import (
	"context"
	"fmt"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/devnet"
)

// ManagedAgent bundles an agent with its client and metadata.
type ManagedAgent struct {
	Name    string
	Agent   *agentsdk.Agent
	Client  *agentsdk.CLIClient
	Role    agentsdk.AgentRole
	Address string // bech32 chain address (from keyring, not SDK key)
}

// AgentPool manages multiple agents sharing a single devnet.
type AgentPool struct {
	Harness *devnet.Harness
	Agents  []*ManagedAgent
	byName  map[string]*ManagedAgent
}

// NewPool creates an empty agent pool bound to a harness.
func NewPool(h *devnet.Harness) *AgentPool {
	return &AgentPool{
		Harness: h,
		byName:  make(map[string]*ManagedAgent),
	}
}

// Add creates a new agent with the given keyring name and role.
// The key must already exist in the harness (test1..test10 are pre-created).
func (p *AgentPool) Add(name string, role agentsdk.AgentRole) (*ManagedAgent, error) {
	agent, client, err := CreateAgent(p.Harness, name, role)
	if err != nil {
		return nil, fmt.Errorf("creating agent %s: %w", name, err)
	}

	ma := &ManagedAgent{
		Name:    name,
		Agent:   agent,
		Client:  client,
		Role:    role,
		Address: p.Harness.KeyAddress(name),
	}
	p.Agents = append(p.Agents, ma)
	p.byName[name] = ma
	return ma, nil
}

// FundAll sends amount ulgm from the "validator" key to every agent in the pool.
// Sends sequentially with 1-block delays to avoid nonce races.
func (p *AgentPool) FundAll(ctx context.Context, amount uint64) error {
	for _, ma := range p.Agents {
		if err := p.Harness.FundAccount(ctx, "validator", ma.Address, amount); err != nil {
			return fmt.Errorf("funding %s: %w", ma.Name, err)
		}
		// Wait ~1 block for tx inclusion before next send
		time.Sleep(agentsdk.BlockTime)
	}
	return nil
}

// RegisterAll registers each agent's on-chain account.
func (p *AgentPool) RegisterAll(ctx context.Context) error {
	for _, ma := range p.Agents {
		if _, err := ma.Client.RegisterAccount(ctx, ""); err != nil {
			// Account may already exist — non-fatal
			_ = err
		}
		time.Sleep(agentsdk.BlockTime)
	}
	return nil
}

// Get returns a managed agent by name.
func (p *AgentPool) Get(name string) *ManagedAgent {
	return p.byName[name]
}

// ByRole returns all agents with the given role.
func (p *AgentPool) ByRole(role agentsdk.AgentRole) []*ManagedAgent {
	var result []*ManagedAgent
	for _, ma := range p.Agents {
		if ma.Role == role {
			result = append(result, ma)
		}
	}
	return result
}
