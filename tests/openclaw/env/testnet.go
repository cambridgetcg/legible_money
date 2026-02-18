//go:build openclaw

package env

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/testnet"
)

// MultiNodeEnv wraps testnet.MultiHarness as an Environment.
type MultiNodeEnv struct {
	harness *testnet.MultiHarness
}

// NewMultiNodeEnv creates an unstarted multi-validator environment.
func NewMultiNodeEnv(t *testing.T, binary string) *MultiNodeEnv {
	return &MultiNodeEnv{
		harness: testnet.NewMultiHarness(t, binary),
	}
}

// MultiHarness returns the underlying testnet.MultiHarness for advanced usage.
func (e *MultiNodeEnv) MultiHarness() *testnet.MultiHarness { return e.harness }

// Start boots the multi-validator testnet.
func (e *MultiNodeEnv) Start(ctx context.Context) error {
	return e.harness.Start(ctx)
}

// StartWithGenesisPatcher boots with a custom genesis patcher.
func (e *MultiNodeEnv) StartWithGenesisPatcher(ctx context.Context, patcher func(string) error) error {
	return e.harness.StartWithGenesisPatcher(ctx, patcher)
}

// --- Environment interface ---

func (e *MultiNodeEnv) Kind() Kind              { return KindTestnet }
func (e *MultiNodeEnv) ChainID() string          { return e.harness.ChainID }
func (e *MultiNodeEnv) RPCAddr() string           { return e.harness.RPCAddr() }
func (e *MultiNodeEnv) RESTAddr() string          { return e.harness.RESTAddr() }
func (e *MultiNodeEnv) GRPCAddr() string          { return e.harness.GRPCAddr() }
func (e *MultiNodeEnv) Binary() string            { return e.harness.Binary }
func (e *MultiNodeEnv) NodeHome() string          { return e.harness.NodeHome() }
func (e *MultiNodeEnv) ValidatorCount() int       { return 3 }

func (e *MultiNodeEnv) ValidatorRPCAddrs() []string {
	return e.harness.ValidatorRPCAddrs()
}

func (e *MultiNodeEnv) NewCLIClient(keyName string) *agentsdk.CLIClient {
	return e.harness.NewCLIClient(keyName)
}

func (e *MultiNodeEnv) KeyAddress(name string) string {
	return e.harness.KeyAddress(name)
}

func (e *MultiNodeEnv) AddKey(name string) (string, error) {
	return e.harness.AddKey(name)
}

func (e *MultiNodeEnv) FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error {
	return e.harness.FundAccount(ctx, fromKey, toAddr, amount)
}

func (e *MultiNodeEnv) WaitForBlock(ctx context.Context, height int64) error {
	return e.harness.WaitForBlock(ctx, height)
}
