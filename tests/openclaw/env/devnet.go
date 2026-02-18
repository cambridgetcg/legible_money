//go:build openclaw

package env

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/devnet"
)

// SingleNodeEnv wraps devnet.Harness as an Environment.
type SingleNodeEnv struct {
	harness *devnet.Harness
}

// NewSingleNodeEnv creates an unstarted single-validator environment.
func NewSingleNodeEnv(t *testing.T, binary string) *SingleNodeEnv {
	return &SingleNodeEnv{
		harness: devnet.NewHarness(t, binary),
	}
}

// Harness returns the underlying devnet.Harness for advanced usage.
func (e *SingleNodeEnv) Harness() *devnet.Harness { return e.harness }

// Start boots the single-validator devnet.
func (e *SingleNodeEnv) Start(ctx context.Context) error {
	return e.harness.Start(ctx)
}

// StartWithGenesisPatcher boots with a custom genesis patcher.
func (e *SingleNodeEnv) StartWithGenesisPatcher(ctx context.Context, patcher func(string) error) error {
	return e.harness.StartWithGenesisPatcher(ctx, devnet.GenesisPatcher(patcher))
}

// --- Environment interface ---

func (e *SingleNodeEnv) Kind() Kind              { return KindDevnet }
func (e *SingleNodeEnv) ChainID() string          { return e.harness.ChainID }
func (e *SingleNodeEnv) RPCAddr() string           { return e.harness.RPCAddr }
func (e *SingleNodeEnv) RESTAddr() string          { return e.harness.RESTAddr }
func (e *SingleNodeEnv) GRPCAddr() string          { return e.harness.GRPCAddr }
func (e *SingleNodeEnv) Binary() string            { return e.harness.Binary }
func (e *SingleNodeEnv) NodeHome() string          { return e.harness.NodeHome }
func (e *SingleNodeEnv) ValidatorCount() int       { return 1 }
func (e *SingleNodeEnv) ValidatorRPCAddrs() []string { return []string{e.harness.RPCAddr} }

func (e *SingleNodeEnv) NewCLIClient(keyName string) *agentsdk.CLIClient {
	return e.harness.NewCLIClient(keyName)
}

func (e *SingleNodeEnv) KeyAddress(name string) string {
	return e.harness.KeyAddress(name)
}

func (e *SingleNodeEnv) AddKey(name string) (string, error) {
	return e.harness.AddKey(name)
}

func (e *SingleNodeEnv) FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error {
	return e.harness.FundAccount(ctx, fromKey, toAddr, amount)
}

func (e *SingleNodeEnv) WaitForBlock(ctx context.Context, height int64) error {
	return e.harness.WaitForBlock(ctx, height)
}
