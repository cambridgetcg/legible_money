//go:build openclaw

// Package env provides an environment abstraction for OpenClaw tests.
// Tests use the Environment interface instead of directly instantiating
// devnet.Harness, enabling seamless switching between single-validator devnet,
// multi-validator testnet, and external chain backends.
package env

import (
	"context"
	"fmt"
	"os"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// Kind identifies the environment backend.
type Kind string

const (
	KindDevnet   Kind = "devnet"   // Single-validator ephemeral chain
	KindTestnet  Kind = "testnet"  // Multi-validator ephemeral chain
	KindExternal Kind = "external" // Pre-existing chain (connected via env vars)
)

// Environment abstracts chain access for OpenClaw tests.
// All test files use this interface instead of raw harness types.
type Environment interface {
	// Kind returns the backend type.
	Kind() Kind
	// ChainID returns the chain identifier.
	ChainID() string
	// RPCAddr returns the CometBFT RPC address (e.g. tcp://localhost:26657).
	RPCAddr() string
	// RESTAddr returns the REST/LCD address (e.g. http://localhost:1317).
	RESTAddr() string
	// GRPCAddr returns the gRPC address (e.g. localhost:9090).
	GRPCAddr() string
	// Binary returns the path to the legbled binary ("" for external).
	Binary() string
	// NodeHome returns the node home directory ("" for external).
	NodeHome() string
	// NewCLIClient creates a CLIClient pointed at this environment.
	NewCLIClient(keyName string) *agentsdk.CLIClient
	// KeyAddress returns the bech32 address for a named key.
	KeyAddress(name string) string
	// AddKey creates a new keyring key and returns its address.
	AddKey(name string) (string, error)
	// FundAccount sends ulgm from one key to an address.
	FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error
	// WaitForBlock waits until the chain reaches the given height.
	WaitForBlock(ctx context.Context, height int64) error
	// ValidatorCount returns the number of validators in this environment.
	ValidatorCount() int
	// ValidatorRPCAddrs returns RPC addresses for all validators.
	ValidatorRPCAddrs() []string
}

// Detect picks an environment backend based on env vars:
//   - $LGM_TESTNET_RPC set   -> ExternalEnv
//   - $LGM_ENV == "testnet"  -> MultiNodeEnv
//   - default                 -> SingleNodeEnv
func Detect(t *testing.T, binary string) Environment {
	t.Helper()

	if rpc := os.Getenv("LGM_TESTNET_RPC"); rpc != "" {
		t.Logf("env.Detect: external chain at %s", rpc)
		return NewExternalEnv(t)
	}

	if os.Getenv("LGM_ENV") == "testnet" {
		t.Log("env.Detect: multi-validator testnet")
		return NewMultiNodeEnv(t, binary)
	}

	t.Log("env.Detect: single-validator devnet")
	return NewSingleNodeEnv(t, binary)
}

// Boot detects the environment, starts it, and returns it ready for use.
func Boot(t *testing.T, binary string, ctx context.Context) Environment {
	t.Helper()
	e := Detect(t, binary)

	switch e := e.(type) {
	case *SingleNodeEnv:
		if err := e.Start(ctx); err != nil {
			t.Fatalf("env.Boot (devnet): %v", err)
		}
	case *MultiNodeEnv:
		if err := e.Start(ctx); err != nil {
			t.Fatalf("env.Boot (testnet): %v", err)
		}
	case *ExternalEnv:
		// no-op: chain already running
	default:
		t.Fatalf("env.Boot: unknown environment type %T", e)
	}

	return e
}

// BootWithGenesisPatcher detects the environment, applies a custom genesis
// patcher, and starts it. Falls back to Boot for external environments.
func BootWithGenesisPatcher(t *testing.T, binary string, ctx context.Context, patcher func(string) error) Environment {
	t.Helper()
	e := Detect(t, binary)

	switch e := e.(type) {
	case *SingleNodeEnv:
		if err := e.StartWithGenesisPatcher(ctx, patcher); err != nil {
			t.Fatalf("env.BootWithGenesisPatcher (devnet): %v", err)
		}
	case *MultiNodeEnv:
		if err := e.StartWithGenesisPatcher(ctx, patcher); err != nil {
			t.Fatalf("env.BootWithGenesisPatcher (testnet): %v", err)
		}
	case *ExternalEnv:
		t.Log("env.BootWithGenesisPatcher: genesis patcher ignored for external environment")
	default:
		t.Fatalf("env.BootWithGenesisPatcher: unknown environment type %T", e)
	}

	return e
}

// RequireMultiValidator skips the test if the environment has fewer than n validators.
func RequireMultiValidator(t *testing.T, e Environment, n int) {
	t.Helper()
	if e.ValidatorCount() < n {
		t.Skipf("requires %d validators, environment has %d", n, e.ValidatorCount())
	}
}

// EnvInfo returns a human-readable summary of the environment for logging.
func EnvInfo(e Environment) string {
	return fmt.Sprintf("kind=%s chain_id=%s validators=%d rpc=%s",
		e.Kind(), e.ChainID(), e.ValidatorCount(), e.RPCAddr())
}
