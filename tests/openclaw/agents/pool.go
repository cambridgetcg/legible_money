//go:build openclaw

package agents

import (
	"context"
	"fmt"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/devnet"
	"github.com/legible-money/legible/tests/openclaw/env"
)

// ManagedAgent bundles an agent with its client and metadata.
type ManagedAgent struct {
	Name    string
	Agent   *agentsdk.Agent
	Client  *agentsdk.CLIClient
	Role    agentsdk.AgentRole
	Address string // bech32 chain address (from keyring, not SDK key)
}

// AgentPool manages multiple agents sharing a single environment.
type AgentPool struct {
	Harness *devnet.Harness // non-nil for legacy NewPool usage
	Env     env.Environment // non-nil for NewPoolFromEnv usage
	Agents  []*ManagedAgent
	byName  map[string]*ManagedAgent
}

// NewPool creates an empty agent pool bound to a harness (legacy).
func NewPool(h *devnet.Harness) *AgentPool {
	return &AgentPool{
		Harness: h,
		byName:  make(map[string]*ManagedAgent),
	}
}

// NewPoolFromEnv creates an empty agent pool bound to an Environment.
func NewPoolFromEnv(e env.Environment) *AgentPool {
	return &AgentPool{
		Env:    e,
		byName: make(map[string]*ManagedAgent),
	}
}

// Add creates a new agent with the given keyring name and role.
// The key must already exist (test1..test10 are pre-created).
func (p *AgentPool) Add(name string, role agentsdk.AgentRole) (*ManagedAgent, error) {
	var agent *agentsdk.Agent
	var client *agentsdk.CLIClient
	var err error
	var addr string

	if p.Env != nil {
		agent, client, err = CreateAgentFromEnv(p.Env, name, role)
		if err != nil {
			return nil, fmt.Errorf("creating agent %s: %w", name, err)
		}
		addr = p.Env.KeyAddress(name)
	} else {
		agent, client, err = CreateAgent(p.Harness, name, role)
		if err != nil {
			return nil, fmt.Errorf("creating agent %s: %w", name, err)
		}
		addr = p.Harness.KeyAddress(name)
	}

	ma := &ManagedAgent{
		Name:    name,
		Agent:   agent,
		Client:  client,
		Role:    role,
		Address: addr,
	}
	p.Agents = append(p.Agents, ma)
	p.byName[name] = ma
	return ma, nil
}

// FundAll sends amount ulgm from the "validator" key to every agent in the pool.
// Sends sequentially with 1-block delays to avoid nonce races.
func (p *AgentPool) FundAll(ctx context.Context, amount uint64) error {
	for _, ma := range p.Agents {
		var err error
		if p.Env != nil {
			err = p.Env.FundAccount(ctx, "validator", ma.Address, amount)
		} else {
			err = p.Harness.FundAccount(ctx, "validator", ma.Address, amount)
		}
		if err != nil {
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
