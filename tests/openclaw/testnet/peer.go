//go:build openclaw

package testnet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// BuildPersistentPeers builds a comma-separated persistent_peers string
// from node IDs and their P2P ports. Format: nodeID@127.0.0.1:port
func BuildPersistentPeers(nodeIDs []string, p2pPorts []int) string {
	peers := make([]string, len(nodeIDs))
	for i, id := range nodeIDs {
		peers[i] = fmt.Sprintf("%s@127.0.0.1:%d", id, p2pPorts[i])
	}
	return strings.Join(peers, ",")
}

// PatchPeers sets the persistent_peers field in config.toml.
func PatchPeers(configPath, peers string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config.toml: %w", err)
	}
	content := string(data)

	// Replace the persistent_peers line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "persistent_peers") && strings.Contains(trimmed, "=") {
			lines[i] = fmt.Sprintf(`persistent_peers = "%s"`, peers)
			break
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
}

// PatchAllowDuplicateIP enables allow_duplicate_ip in config.toml.
// Required when all validators run on 127.0.0.1.
func PatchAllowDuplicateIP(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config.toml: %w", err)
	}
	content := string(data)
	content = strings.ReplaceAll(content,
		"allow_duplicate_ip = false",
		"allow_duplicate_ip = true")
	return os.WriteFile(configPath, []byte(content), 0644)
}

// currentHeightOnPort queries the RPC status endpoint at the given port.
func currentHeightOnPort(rpcPort int) (int64, error) {
	url := fmt.Sprintf("http://localhost:%d/status", rpcPort)
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

// waitForBlockOnPort waits for a specific port's node to reach the target height.
func waitForBlockOnPort(ctx context.Context, rpcPort int, height int64) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for height %d on port %d: %w", height, rpcPort, ctx.Err())
		case <-ticker.C:
			h, err := currentHeightOnPort(rpcPort)
			if err == nil && h >= height {
				return nil
			}
		}
	}
}
