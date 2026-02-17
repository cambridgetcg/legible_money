//go:build openclaw

package devnet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

const (
	validatorBalance = "1000000000000ulgm" // 1M LGM
	testBalance      = "100000000000ulgm"  // 100K LGM
	selfDelegation   = "10000000000ulgm"   // 10K LGM
	keyring          = "test"
	moniker          = "openclaw-validator"
)

// Harness manages an ephemeral single-validator devnet for testing.
type Harness struct {
	T        *testing.T
	Binary   string // Path to compiled legbled binary
	NodeHome string // Unique temp dir per harness
	ChainID  string
	RPCAddr  string // tcp://localhost:{port}
	RESTAddr string // http://localhost:{port}
	GRPCAddr string // localhost:{port}
	Cmd      *exec.Cmd

	keyAddresses map[string]string
	rpcPort      int
	logFile      *os.File
}

// NewHarness creates a harness bound to a test. Call Start() to boot the chain.
func NewHarness(t *testing.T, binary string) *Harness {
	t.Helper()
	return &Harness{
		T:            t,
		Binary:       binary,
		NodeHome:     t.TempDir(),
		ChainID:      fmt.Sprintf("lgm-oca-%d", time.Now().UnixNano()%100000),
		keyAddresses: make(map[string]string),
	}
}

// Start boots the devnet: init, keys, genesis, start, wait for blocks.
func (h *Harness) Start(ctx context.Context) error {
	h.T.Helper()

	// 1. Allocate 6 ports: RPC, P2P, pprof, REST, gRPC, gRPC-web
	ports, err := AllocatePorts(6)
	if err != nil {
		return fmt.Errorf("allocating ports: %w", err)
	}
	h.rpcPort = ports[0]
	p2pPort := ports[1]
	pprofPort := ports[2]
	restPort := ports[3]
	grpcPort := ports[4]
	grpcWebPort := ports[5]

	h.RPCAddr = fmt.Sprintf("tcp://localhost:%d", h.rpcPort)
	h.RESTAddr = fmt.Sprintf("http://localhost:%d", restPort)
	h.GRPCAddr = fmt.Sprintf("localhost:%d", grpcPort)

	// 2. Init chain
	if err := h.run(ctx, "init", moniker, "--chain-id", h.ChainID, "--default-denom", "ulgm"); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// 3. Create keys: validator + 10 test agents
	keyNames := []string{"validator"}
	for i := 1; i <= 10; i++ {
		keyNames = append(keyNames, fmt.Sprintf("test%d", i))
	}
	for _, name := range keyNames {
		if err := h.addKey(name); err != nil {
			return fmt.Errorf("adding key %s: %w", name, err)
		}
	}

	// 4. Add genesis accounts
	if err := h.run(ctx, "add-genesis-account", h.keyAddresses["validator"], validatorBalance); err != nil {
		return fmt.Errorf("add-genesis-account validator: %w", err)
	}
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("test%d", i)
		if err := h.run(ctx, "add-genesis-account", h.keyAddresses[name], testBalance); err != nil {
			return fmt.Errorf("add-genesis-account %s: %w", name, err)
		}
	}

	// 5. Patch genesis
	genesisPath := filepath.Join(h.NodeHome, "config", "genesis.json")
	if err := PatchGenesis(genesisPath); err != nil {
		return fmt.Errorf("patching genesis: %w", err)
	}

	// 6. Patch configs with dynamic ports
	configPath := filepath.Join(h.NodeHome, "config", "config.toml")
	appPath := filepath.Join(h.NodeHome, "config", "app.toml")
	if err := PatchCometConfig(configPath, h.rpcPort, p2pPort, pprofPort); err != nil {
		return fmt.Errorf("patching config.toml: %w", err)
	}
	if err := PatchAppConfig(appPath, restPort, grpcPort, grpcWebPort); err != nil {
		return fmt.Errorf("patching app.toml: %w", err)
	}

	// 7. Gentx
	if err := h.run(ctx, "gentx", "validator", selfDelegation,
		"--chain-id", h.ChainID,
		"--keyring-backend", keyring); err != nil {
		return fmt.Errorf("gentx: %w", err)
	}

	// 8. Collect gentxs
	if err := h.run(ctx, "collect-gentxs"); err != nil {
		return fmt.Errorf("collect-gentxs: %w", err)
	}

	// 9. Validate genesis
	if err := h.run(ctx, "validate-genesis"); err != nil {
		return fmt.Errorf("validate-genesis: %w", err)
	}

	// 10. Start node as background process
	logPath := filepath.Join(h.NodeHome, "node.log")
	h.logFile, _ = os.Create(logPath)

	cmd := exec.Command(h.Binary, "start",
		"--home", h.NodeHome,
		"--minimum-gas-prices", "0.025ulgm")
	if h.logFile != nil {
		cmd.Stdout = h.logFile
		cmd.Stderr = h.logFile
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting node: %w", err)
	}
	h.Cmd = cmd

	// 10a. Quick check: did the process exit immediately?
	earlyExit := make(chan error, 1)
	go func() { earlyExit <- cmd.Wait() }()
	select {
	case waitErr := <-earlyExit:
		// Process exited before we even started polling
		if h.logFile != nil {
			h.logFile.Sync()
		}
		h.dumpFullLogs()
		h.T.Logf("Node binary: %s", h.Binary)
		h.T.Logf("Node home: %s", h.NodeHome)
		h.T.Logf("Start args: start --home %s --minimum-gas-prices 0.025ulgm", h.NodeHome)
		return fmt.Errorf("node exited immediately: %w", waitErr)
	case <-time.After(5 * time.Second):
		// Process still running — good
	}

	// 11. Wait for height >= 2
	if err := h.WaitForBlock(ctx, 2); err != nil {
		// Dump last logs on failure
		h.dumpFullLogs()
		return fmt.Errorf("waiting for blocks: %w", err)
	}

	// 12. Register cleanup
	h.T.Cleanup(func() { h.Stop() })

	h.T.Logf("Devnet started: chain_id=%s rpc=:%d rest=:%d", h.ChainID, h.rpcPort, restPort)
	return nil
}

// Stop sends SIGTERM to the node, waits briefly, then SIGKILL if needed.
func (h *Harness) Stop() {
	if h.Cmd == nil || h.Cmd.Process == nil {
		return
	}

	_ = h.Cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- h.Cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = h.Cmd.Process.Kill()
		<-done
	}

	if h.logFile != nil {
		h.logFile.Close()
	}
}

// WaitForBlock polls the RPC status endpoint until the chain reaches the target height.
func (h *Harness) WaitForBlock(ctx context.Context, height int64) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for block %d: %w", height, ctx.Err())
		case <-ticker.C:
			current, err := h.currentHeight()
			if err != nil {
				continue // Node not ready yet
			}
			if current >= height {
				return nil
			}
		}
	}
}

// AddKey creates a new keyring key and caches its address.
// Use this to add keys beyond the default 10 test keys.
func (h *Harness) AddKey(name string) (string, error) {
	if err := h.addKey(name); err != nil {
		return "", err
	}
	return h.keyAddresses[name], nil
}

// KeyAddress returns the bech32 address for a named key.
func (h *Harness) KeyAddress(name string) string {
	addr, ok := h.keyAddresses[name]
	if ok {
		return addr
	}
	// Fallback: query the keyring
	cmd := exec.Command(h.Binary, "keys", "show", name, "-a",
		"--keyring-backend", keyring,
		"--home", h.NodeHome)
	out, err := cmd.Output()
	if err != nil {
		h.T.Fatalf("KeyAddress(%s): %v", name, err)
	}
	addr = strings.TrimSpace(string(out))
	h.keyAddresses[name] = addr
	return addr
}

// FundAccount sends ulgm from one keyring key to an address.
func (h *Harness) FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error {
	args := []string{"tx", "bank", "send", fromKey, toAddr,
		fmt.Sprintf("%dulgm", amount),
		"--keyring-backend", keyring,
		"--chain-id", h.ChainID,
		"--node", h.RPCAddr,
		"--gas-prices", "0.025ulgm",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"-y",
		"--home", h.NodeHome,
	}
	cmd := exec.CommandContext(ctx, h.Binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("funding %s: %s: %w", toAddr, string(out), err)
	}
	return nil
}

// NewCLIClient creates a CLIClient pointed at this harness for the given key.
func (h *Harness) NewCLIClient(keyName string) *agentsdk.CLIClient {
	return &agentsdk.CLIClient{
		Binary:   h.Binary,
		KeyName:  keyName,
		Keyring:  keyring,
		Home:     h.NodeHome,
		Node:     h.RPCAddr,
		RESTAddr: h.RESTAddr,
		ChainID:  h.ChainID,
		GasPrice: 1,
	}
}

// --- Internal helpers ---

// run executes a legbled command with --home flag.
func (h *Harness) run(ctx context.Context, args ...string) error {
	fullArgs := append(args, "--home", h.NodeHome)
	cmd := exec.CommandContext(ctx, h.Binary, fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

// addKey creates a key in the test keyring and caches the address.
func (h *Harness) addKey(name string) error {
	// Create the key
	cmd := exec.Command(h.Binary, "keys", "add", name,
		"--keyring-backend", keyring,
		"--home", h.NodeHome)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(out), err)
	}

	// Read address via keys show -a
	showCmd := exec.Command(h.Binary, "keys", "show", name, "-a",
		"--keyring-backend", keyring,
		"--home", h.NodeHome)
	out, err := showCmd.Output()
	if err != nil {
		return fmt.Errorf("reading address: %w", err)
	}
	h.keyAddresses[name] = strings.TrimSpace(string(out))
	return nil
}

// currentHeight queries the RPC status endpoint for the latest block height.
func (h *Harness) currentHeight() (int64, error) {
	url := fmt.Sprintf("http://localhost:%d/status", h.rpcPort)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var status struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return 0, err
	}

	return strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
}

// dumpLogs prints the last 50 lines of node logs on failure.
func (h *Harness) dumpLogs() {
	logPath := filepath.Join(h.NodeHome, "node.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	start := 0
	if len(lines) > 50 {
		start = len(lines) - 50
	}
	h.T.Logf("=== Last node logs ===\n%s", strings.Join(lines[start:], "\n"))
}

// dumpFullLogs prints the first 100 lines + last 20 lines of node logs.
func (h *Harness) dumpFullLogs() {
	logPath := filepath.Join(h.NodeHome, "node.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		h.T.Logf("Could not read log file: %v", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	h.T.Logf("=== Node log (%d lines total) ===", len(lines))
	if len(lines) <= 120 {
		h.T.Logf("%s", string(data))
	} else {
		h.T.Logf("=== FIRST 100 lines ===\n%s", strings.Join(lines[:100], "\n"))
		h.T.Logf("=== LAST 20 lines ===\n%s", strings.Join(lines[len(lines)-20:], "\n"))
	}
}
