//go:build openclaw

package env

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ExternalEnv connects to a pre-existing chain via environment variables.
// No process management — the chain is already running.
//
// Required env vars:
//
//	$LGM_TESTNET_RPC     — CometBFT RPC (e.g. tcp://localhost:26657)
//	$LGM_TESTNET_REST    — REST/LCD (e.g. http://localhost:1317)
//	$LGM_TESTNET_CHAIN_ID — Chain ID
//
// Optional:
//
//	$LGM_TESTNET_GRPC    — gRPC address
//	$LGM_TESTNET_HOME    — Node home for keyring
//	$LGM_TESTNET_BINARY  — Path to legbled binary
type ExternalEnv struct {
	t       *testing.T
	rpc     string
	rest    string
	grpc    string
	chainID string
	binary  string
	home    string
}

// NewExternalEnv reads connection info from environment variables.
func NewExternalEnv(t *testing.T) *ExternalEnv {
	t.Helper()

	rpc := os.Getenv("LGM_TESTNET_RPC")
	if rpc == "" {
		t.Fatal("ExternalEnv requires $LGM_TESTNET_RPC")
	}
	rest := os.Getenv("LGM_TESTNET_REST")
	if rest == "" {
		t.Fatal("ExternalEnv requires $LGM_TESTNET_REST")
	}
	chainID := os.Getenv("LGM_TESTNET_CHAIN_ID")
	if chainID == "" {
		t.Fatal("ExternalEnv requires $LGM_TESTNET_CHAIN_ID")
	}

	return &ExternalEnv{
		t:       t,
		rpc:     rpc,
		rest:    rest,
		grpc:    os.Getenv("LGM_TESTNET_GRPC"),
		chainID: chainID,
		binary:  os.Getenv("LGM_TESTNET_BINARY"),
		home:    os.Getenv("LGM_TESTNET_HOME"),
	}
}

func (e *ExternalEnv) Kind() Kind     { return KindExternal }
func (e *ExternalEnv) ChainID() string { return e.chainID }
func (e *ExternalEnv) RPCAddr() string  { return e.rpc }
func (e *ExternalEnv) RESTAddr() string { return e.rest }
func (e *ExternalEnv) GRPCAddr() string { return e.grpc }
func (e *ExternalEnv) Binary() string   { return e.binary }
func (e *ExternalEnv) NodeHome() string { return e.home }

func (e *ExternalEnv) NewCLIClient(keyName string) *agentsdk.CLIClient {
	return &agentsdk.CLIClient{
		Binary:   e.binary,
		KeyName:  keyName,
		Keyring:  "test",
		Home:     e.home,
		Node:     e.rpc,
		RESTAddr: e.rest,
		ChainID:  e.chainID,
		GasPrice: 1,
	}
}

func (e *ExternalEnv) KeyAddress(name string) string {
	if e.binary == "" || e.home == "" {
		e.t.Fatalf("ExternalEnv.KeyAddress requires $LGM_TESTNET_BINARY and $LGM_TESTNET_HOME")
	}
	cmd := exec.Command(e.binary, "keys", "show", name, "-a",
		"--keyring-backend", "test",
		"--home", e.home)
	out, err := cmd.Output()
	if err != nil {
		e.t.Fatalf("ExternalEnv.KeyAddress(%s): %v", name, err)
	}
	return strings.TrimSpace(string(out))
}

func (e *ExternalEnv) AddKey(name string) (string, error) {
	if e.binary == "" || e.home == "" {
		return "", fmt.Errorf("ExternalEnv.AddKey requires $LGM_TESTNET_BINARY and $LGM_TESTNET_HOME")
	}
	cmd := exec.Command(e.binary, "keys", "add", name,
		"--keyring-backend", "test",
		"--home", e.home)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: %w", string(out), err)
	}
	return e.KeyAddress(name), nil
}

func (e *ExternalEnv) FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error {
	if e.binary == "" || e.home == "" {
		return fmt.Errorf("ExternalEnv.FundAccount requires $LGM_TESTNET_BINARY and $LGM_TESTNET_HOME")
	}
	args := []string{"tx", "bank", "send", fromKey, toAddr,
		fmt.Sprintf("%dulgm", amount),
		"--keyring-backend", "test",
		"--chain-id", e.chainID,
		"--node", e.rpc,
		"--gas-prices", "0.025ulgm",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"-y",
		"--home", e.home,
	}
	cmd := exec.CommandContext(ctx, e.binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("funding %s: %s: %w", toAddr, string(out), err)
	}
	return nil
}

func (e *ExternalEnv) WaitForBlock(ctx context.Context, height int64) error {
	client := e.NewCLIClient("validator")
	// Delegate to query-based wait
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		status, err := client.ChainStatus(ctx)
		if err == nil && status.LatestHeight >= height {
			return nil
		}
	}
}

// ValidatorCount queries the staking module for active validators.
func (e *ExternalEnv) ValidatorCount() int {
	url := fmt.Sprintf("%s/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED", e.rest)
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}
	var result struct {
		Validators []json.RawMessage `json:"validators"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0
	}
	return len(result.Validators)
}

func (e *ExternalEnv) ValidatorRPCAddrs() []string {
	return []string{e.rpc}
}
