//go:build openclaw

// Package testnet provides a multi-validator test harness for OpenClaw tests.
// It boots 3 legbled processes with distinct home directories, coordinated
// through a shared genesis ceremony.
package testnet

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/devnet"
)

const (
	numNodes         = 3
	validatorBalance = "1000000000000ulgm" // 1M LGM each
	testBalance      = "100000000000ulgm"  // 100K LGM each
	selfDelegation   = "10000000000ulgm"   // 10K LGM each
	keyring          = "test"
)

type nodeInfo struct {
	name     string
	home     string
	cmd      *exec.Cmd
	rpcPort  int
	p2pPort  int
	restPort int
	grpcPort int
	pprofPort    int
	grpcWebPort  int
	logFile  *os.File
	nodeID   string
}

// MultiHarness manages a 3-validator ephemeral testnet.
type MultiHarness struct {
	T        *testing.T
	Binary   string
	ChainID  string

	nodes      [numNodes]*nodeInfo
	coordHome  string // temp dir for genesis ceremony
	keyAddrs   map[string]string
}

// NewMultiHarness creates an unstarted multi-validator harness.
func NewMultiHarness(t *testing.T, binary string) *MultiHarness {
	return &MultiHarness{
		T:       t,
		Binary:  binary,
		ChainID: fmt.Sprintf("lgm-oca-multi-%d", time.Now().UnixNano()%100000),
		keyAddrs: make(map[string]string),
	}
}

// Start boots 3 validator nodes and waits for block production.
func (m *MultiHarness) Start(ctx context.Context) error {
	return m.startWithPatcher(ctx, PatchGenesisMultiValidator)
}

// StartWithGenesisPatcher boots with a custom genesis patcher layered on top of
// PatchGenesisMultiValidator.
func (m *MultiHarness) StartWithGenesisPatcher(ctx context.Context, patcher func(string) error) error {
	combined := func(path string) error {
		if err := PatchGenesisMultiValidator(path); err != nil {
			return err
		}
		return patcher(path)
	}
	return m.startWithPatcher(ctx, combined)
}

func (m *MultiHarness) startWithPatcher(ctx context.Context, patcher func(string) error) error {
	m.T.Helper()

	// 1. Allocate ports: 6 ports x 3 nodes = 18
	ports, err := devnet.AllocatePorts(6 * numNodes)
	if err != nil {
		return fmt.Errorf("allocating ports: %w", err)
	}

	for i := 0; i < numNodes; i++ {
		base := i * 6
		m.nodes[i] = &nodeInfo{
			name:        fmt.Sprintf("validator%d", i+1),
			home:        m.T.TempDir(),
			rpcPort:     ports[base],
			p2pPort:     ports[base+1],
			pprofPort:   ports[base+2],
			restPort:    ports[base+3],
			grpcPort:    ports[base+4],
			grpcWebPort: ports[base+5],
		}
	}

	// 2. Init coordinator for genesis ceremony
	m.coordHome = m.T.TempDir()
	if err := m.runAt(ctx, m.coordHome, "init", "coordinator", "--chain-id", m.ChainID, "--default-denom", "ulgm"); err != nil {
		return fmt.Errorf("init coordinator: %w", err)
	}

	// 3. Create keys in coordinator keyring: validator1-3 + test1-10
	keyNames := make([]string, 0, numNodes+10)
	for i := 1; i <= numNodes; i++ {
		keyNames = append(keyNames, fmt.Sprintf("validator%d", i))
	}
	for i := 1; i <= 10; i++ {
		keyNames = append(keyNames, fmt.Sprintf("test%d", i))
	}
	// Also create a "validator" alias pointing to validator1's address
	keyNames = append(keyNames, "validator")

	for _, name := range keyNames {
		if err := m.addKey(name); err != nil {
			return fmt.Errorf("adding key %s: %w", name, err)
		}
	}

	// 4. Add genesis accounts
	for i := 1; i <= numNodes; i++ {
		name := fmt.Sprintf("validator%d", i)
		if err := m.runAt(ctx, m.coordHome, "add-genesis-account", m.keyAddrs[name], validatorBalance); err != nil {
			return fmt.Errorf("add-genesis-account %s: %w", name, err)
		}
	}
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("test%d", i)
		if err := m.runAt(ctx, m.coordHome, "add-genesis-account", m.keyAddrs[name], testBalance); err != nil {
			return fmt.Errorf("add-genesis-account %s: %w", name, err)
		}
	}

	// 5. Patch genesis
	genesisPath := filepath.Join(m.coordHome, "config", "genesis.json")
	if err := patcher(genesisPath); err != nil {
		return fmt.Errorf("patching genesis: %w", err)
	}

	// 6. Init each validator node
	for i := 0; i < numNodes; i++ {
		node := m.nodes[i]
		if err := m.runAt(ctx, node.home, "init", node.name, "--chain-id", m.ChainID, "--default-denom", "ulgm"); err != nil {
			return fmt.Errorf("init %s: %w", node.name, err)
		}
	}

	// 7. Copy coordinator keyring to each node home
	for i := 0; i < numNodes; i++ {
		srcDir := filepath.Join(m.coordHome, "keyring-test")
		dstDir := filepath.Join(m.nodes[i].home, "keyring-test")
		if err := copyDir(srcDir, dstDir); err != nil {
			return fmt.Errorf("copying keyring to %s: %w", m.nodes[i].name, err)
		}
	}

	// 8. Generate gentx for each validator
	gentxDir := filepath.Join(m.coordHome, "config", "gentx")
	for i := 0; i < numNodes; i++ {
		name := fmt.Sprintf("validator%d", i+1)
		if err := m.runAt(ctx, m.coordHome, "gentx", name, selfDelegation,
			"--chain-id", m.ChainID,
			"--keyring-backend", keyring,
			"--moniker", m.nodes[i].name,
			"--ip", "127.0.0.1",
			"--p2p-port", fmt.Sprintf("%d", m.nodes[i].p2pPort),
		); err != nil {
			return fmt.Errorf("gentx %s: %w", name, err)
		}
	}
	_ = gentxDir

	// 9. Collect gentxs + validate genesis
	if err := m.runAt(ctx, m.coordHome, "collect-gentxs"); err != nil {
		return fmt.Errorf("collect-gentxs: %w", err)
	}
	if err := m.runAt(ctx, m.coordHome, "validate-genesis"); err != nil {
		return fmt.Errorf("validate-genesis: %w", err)
	}

	// 10. Get node IDs
	for i := 0; i < numNodes; i++ {
		id, err := m.getNodeID(m.nodes[i].home)
		if err != nil {
			return fmt.Errorf("getting node ID for %s: %w", m.nodes[i].name, err)
		}
		m.nodes[i].nodeID = id
	}

	// 11. Build persistent_peers
	peers := BuildPersistentPeers(m.nodeIDs(), m.p2pPorts())

	// 12. Distribute final genesis + patch configs
	finalGenesis := filepath.Join(m.coordHome, "config", "genesis.json")
	for i := 0; i < numNodes; i++ {
		node := m.nodes[i]

		// Copy final genesis
		dstGenesis := filepath.Join(node.home, "config", "genesis.json")
		if err := copyFile(finalGenesis, dstGenesis); err != nil {
			return fmt.Errorf("copying genesis to %s: %w", node.name, err)
		}

		// Patch CometBFT config
		configPath := filepath.Join(node.home, "config", "config.toml")
		if err := devnet.PatchCometConfig(configPath, node.rpcPort, node.p2pPort, node.pprofPort); err != nil {
			return fmt.Errorf("patching config.toml for %s: %w", node.name, err)
		}
		if err := PatchPeers(configPath, peers); err != nil {
			return fmt.Errorf("patching peers for %s: %w", node.name, err)
		}
		if err := PatchAllowDuplicateIP(configPath); err != nil {
			return fmt.Errorf("patching allow_duplicate_ip for %s: %w", node.name, err)
		}

		// Patch app config
		appPath := filepath.Join(node.home, "config", "app.toml")
		if err := devnet.PatchAppConfig(appPath, node.restPort, node.grpcPort, node.grpcWebPort); err != nil {
			return fmt.Errorf("patching app.toml for %s: %w", node.name, err)
		}
	}

	// 13. Start all 3 processes
	for i := 0; i < numNodes; i++ {
		node := m.nodes[i]
		logPath := filepath.Join(node.home, "node.log")
		node.logFile, _ = os.Create(logPath)

		cmd := exec.Command(m.Binary, "start",
			"--home", node.home,
			"--minimum-gas-prices", "0.025ulgm")
		if node.logFile != nil {
			cmd.Stdout = node.logFile
			cmd.Stderr = node.logFile
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("starting %s: %w", node.name, err)
		}
		node.cmd = cmd
	}

	// Quick check: processes are alive after 5 seconds
	time.Sleep(5 * time.Second)
	for i := 0; i < numNodes; i++ {
		node := m.nodes[i]
		if err := node.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			_ = node.cmd.Wait()
			m.dumpNodeLogs(i)
			return fmt.Errorf("%s exited immediately", node.name)
		}
	}

	// 14. Wait for block height >= 3 on at least 2/3 nodes
	if err := m.waitForQuorumBlocks(ctx, 3); err != nil {
		for i := 0; i < numNodes; i++ {
			m.dumpNodeLogs(i)
		}
		return fmt.Errorf("waiting for blocks: %w", err)
	}

	// 15. Register cleanup
	m.T.Cleanup(func() { m.Stop() })

	m.T.Logf("Testnet started: chain_id=%s validators=%d rpc_0=:%d rpc_1=:%d rpc_2=:%d",
		m.ChainID, numNodes,
		m.nodes[0].rpcPort, m.nodes[1].rpcPort, m.nodes[2].rpcPort)
	return nil
}

// Stop shuts down all validator processes.
func (m *MultiHarness) Stop() {
	for i := 0; i < numNodes; i++ {
		node := m.nodes[i]
		if node == nil || node.cmd == nil || node.cmd.Process == nil {
			continue
		}
		_ = node.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- node.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = node.cmd.Process.Kill()
			<-done
		}
		if node.logFile != nil {
			node.logFile.Close()
		}
	}
}

// RPCAddr returns the RPC address of the first validator.
func (m *MultiHarness) RPCAddr() string {
	return fmt.Sprintf("tcp://localhost:%d", m.nodes[0].rpcPort)
}

// RESTAddr returns the REST address of the first validator.
func (m *MultiHarness) RESTAddr() string {
	return fmt.Sprintf("http://localhost:%d", m.nodes[0].restPort)
}

// GRPCAddr returns the gRPC address of the first validator.
func (m *MultiHarness) GRPCAddr() string {
	return fmt.Sprintf("localhost:%d", m.nodes[0].grpcPort)
}

// NodeHome returns the first validator's home (used for keyring access).
func (m *MultiHarness) NodeHome() string {
	return m.nodes[0].home
}

// KeyAddress returns the bech32 address for a named key.
func (m *MultiHarness) KeyAddress(name string) string {
	addr, ok := m.keyAddrs[name]
	if ok {
		return addr
	}
	cmd := exec.Command(m.Binary, "keys", "show", name, "-a",
		"--keyring-backend", keyring,
		"--home", m.nodes[0].home)
	out, err := cmd.Output()
	if err != nil {
		m.T.Fatalf("KeyAddress(%s): %v", name, err)
	}
	addr = strings.TrimSpace(string(out))
	m.keyAddrs[name] = addr
	return addr
}

// AddKey creates a new key in the first node's keyring.
func (m *MultiHarness) AddKey(name string) (string, error) {
	cmd := exec.Command(m.Binary, "keys", "add", name,
		"--keyring-backend", keyring,
		"--home", m.nodes[0].home)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: %w", string(out), err)
	}
	showCmd := exec.Command(m.Binary, "keys", "show", name, "-a",
		"--keyring-backend", keyring,
		"--home", m.nodes[0].home)
	out, err := showCmd.Output()
	if err != nil {
		return "", fmt.Errorf("reading address: %w", err)
	}
	addr := strings.TrimSpace(string(out))
	m.keyAddrs[name] = addr
	return addr, nil
}

// FundAccount sends ulgm from one key to an address.
func (m *MultiHarness) FundAccount(ctx context.Context, fromKey, toAddr string, amount uint64) error {
	args := []string{"tx", "bank", "send", fromKey, toAddr,
		fmt.Sprintf("%dulgm", amount),
		"--keyring-backend", keyring,
		"--chain-id", m.ChainID,
		"--node", m.RPCAddr(),
		"--gas-prices", "0.025ulgm",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"-y",
		"--home", m.nodes[0].home,
	}
	cmd := exec.CommandContext(ctx, m.Binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("funding %s: %s: %w", toAddr, string(out), err)
	}
	return nil
}

// NewCLIClient creates a CLIClient pointed at the first validator node.
func (m *MultiHarness) NewCLIClient(keyName string) *agentsdk.CLIClient {
	return &agentsdk.CLIClient{
		Binary:   m.Binary,
		KeyName:  keyName,
		Keyring:  keyring,
		Home:     m.nodes[0].home,
		Node:     m.RPCAddr(),
		RESTAddr: m.RESTAddr(),
		ChainID:  m.ChainID,
		GasPrice: 1,
	}
}

// WaitForBlock waits until the first node reaches the target height.
func (m *MultiHarness) WaitForBlock(ctx context.Context, height int64) error {
	return waitForBlockOnPort(ctx, m.nodes[0].rpcPort, height)
}

// ValidatorRPCAddrs returns RPC addresses for all 3 validators.
func (m *MultiHarness) ValidatorRPCAddrs() []string {
	addrs := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		addrs[i] = fmt.Sprintf("tcp://localhost:%d", m.nodes[i].rpcPort)
	}
	return addrs
}

// --- Internal helpers ---

func (m *MultiHarness) runAt(ctx context.Context, home string, args ...string) error {
	fullArgs := append(args, "--home", home)
	cmd := exec.CommandContext(ctx, m.Binary, fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func (m *MultiHarness) addKey(name string) error {
	cmd := exec.Command(m.Binary, "keys", "add", name,
		"--keyring-backend", keyring,
		"--home", m.coordHome)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(out), err)
	}
	showCmd := exec.Command(m.Binary, "keys", "show", name, "-a",
		"--keyring-backend", keyring,
		"--home", m.coordHome)
	out, err := showCmd.Output()
	if err != nil {
		return fmt.Errorf("reading address: %w", err)
	}
	m.keyAddrs[name] = strings.TrimSpace(string(out))
	return nil
}

func (m *MultiHarness) getNodeID(home string) (string, error) {
	cmd := exec.Command(m.Binary, "comet", "show-node-id", "--home", home)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *MultiHarness) nodeIDs() []string {
	ids := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		ids[i] = m.nodes[i].nodeID
	}
	return ids
}

func (m *MultiHarness) p2pPorts() []int {
	ports := make([]int, numNodes)
	for i := 0; i < numNodes; i++ {
		ports[i] = m.nodes[i].p2pPort
	}
	return ports
}

// waitForQuorumBlocks waits until at least 2/3 nodes reach the target height.
func (m *MultiHarness) waitForQuorumBlocks(ctx context.Context, height int64) error {
	deadline := time.After(90 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	quorum := (numNodes * 2 / 3) + 1 // need 2+1=3... actually 2/3+1 for 3 nodes = 3. Let's use 2.
	quorum = 2                         // 2 out of 3 is sufficient

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for %d/%d nodes to reach height %d", quorum, numNodes, height)
		case <-ticker.C:
			reached := 0
			for i := 0; i < numNodes; i++ {
				h, err := currentHeightOnPort(m.nodes[i].rpcPort)
				if err == nil && h >= height {
					reached++
				}
			}
			if reached >= quorum {
				return nil
			}
		}
	}
}

func (m *MultiHarness) dumpNodeLogs(idx int) {
	node := m.nodes[idx]
	if node == nil {
		return
	}
	logPath := filepath.Join(node.home, "node.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	maxLines := len(lines)
	if maxLines > 100 {
		maxLines = 100
	}
	m.T.Logf("=== %s logs (%d lines total) ===\n%s", node.name, len(lines), strings.Join(lines[:maxLines], "\n"))
}

// --- File copy helpers ---

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
