//go:build openclaw

package agents

import (
	"path/filepath"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/devnet"
)

// CreateAgent creates an Agent SDK instance wired to the harness for a given keyring key.
// The keyring key must already exist in the harness (created during Start or via AddKey).
func CreateAgent(h *devnet.Harness, keyName string, role agentsdk.AgentRole) (*agentsdk.Agent, *agentsdk.CLIClient, error) {
	// Agent config pointing at the harness devnet
	cfg := agentsdk.AgentConfig{
		NodeRPC:  h.RPCAddr,
		ChainID:  h.ChainID,
		KeyDir:   filepath.Join(h.NodeHome, "agent-keys", keyName),
		GasPrice: 1,
		Role:     role,
		Domains:  []string{"general"},
	}

	agent := agentsdk.New(cfg)

	// Generate an SDK-level key for salt management (commit-reveal).
	// This is separate from the chain keyring key used by CLIClient.
	if err := agent.GenerateKey(keyName); err != nil {
		return nil, nil, err
	}

	// Wire CLIClient as both query and tx backend
	client := h.NewCLIClient(keyName)
	agent.SetTxClient(client)
	agent.SetQueryClient(client)

	return agent, client, nil
}
