// workspace.go provides PicoClaw workspace integration for Agent Home sync.
//
// PicoClaw agents have a workspace directory structure:
//   workspace/
//     HEARTBEAT.md      — Periodic task definitions
//     skills/           — Tool scripts
//     memory/           — Agent memory (synced to Agent Home via IPFS CID)
//     config.json       — Agent configuration
//
// This file handles the workspace <-> Agent Home bidirectional sync.

package picoclaw

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceConfig describes the PicoClaw workspace layout.
type WorkspaceConfig struct {
	// WorkspaceDir is the root of the PicoClaw workspace.
	WorkspaceDir string `json:"workspace_dir"`

	// AgentName is the display name for the agent.
	AgentName string `json:"agent_name"`

	// HomeID is the on-chain Agent Home ID (set after creation).
	HomeID string `json:"home_id"`

	// KeyName is the keyring key name for signing transactions.
	KeyName string `json:"key_name"`

	// Domains the agent operates in.
	Domains []string `json:"domains"`

	// Role is the primary agent role.
	Role string `json:"role"`

	// LastSyncCID is the last IPFS CID synced to Agent Home.
	LastSyncCID string `json:"last_sync_cid"`

	// LastSyncTime is when the last sync occurred.
	LastSyncTime string `json:"last_sync_time"`
}

// InitWorkspace creates the PicoClaw workspace directory structure
// with LGM-specific configuration and skill scripts.
func InitWorkspace(dir, agentName, keyName string, domains []string) error {
	dirs := []string{
		filepath.Join(dir, "skills"),
		filepath.Join(dir, "memory"),
		filepath.Join(dir, "memory", "facts"),
		filepath.Join(dir, "memory", "verifications"),
		filepath.Join(dir, "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write config
	cfg := WorkspaceConfig{
		WorkspaceDir: dir,
		AgentName:    agentName,
		HomeID:       "",
		KeyName:      keyName,
		Domains:      domains,
		Role:         "verifier",
	}
	cfgData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), cfgData, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Write HEARTBEAT.md
	heartbeat := generateHeartbeatMD(keyName)
	if err := os.WriteFile(filepath.Join(dir, "HEARTBEAT.md"), []byte(heartbeat), 0644); err != nil {
		return fmt.Errorf("writing HEARTBEAT.md: %w", err)
	}

	// Write environment script
	envScript := generateEnvScript(keyName)
	if err := os.WriteFile(filepath.Join(dir, "skills", "lgm-env.sh"), []byte(envScript), 0755); err != nil {
		return fmt.Errorf("writing lgm-env.sh: %w", err)
	}

	return nil
}

// ComputeMemoryCID computes a content identifier for the memory directory.
// In production this would upload to IPFS; for devnet we use a content hash.
func ComputeMemoryCID(memoryDir string) (string, error) {
	h := sha256.New()

	err := filepath.Walk(memoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(memoryDir, path)
		h.Write([]byte(relPath))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking memory dir: %w", err)
	}

	// Use a bafk... prefix to resemble CIDv1 (base32 would be proper)
	hash := hex.EncodeToString(h.Sum(nil))
	return "bafk" + hash[:56], nil
}

// LoadWorkspaceConfig reads config from a workspace directory.
func LoadWorkspaceConfig(dir string) (*WorkspaceConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg WorkspaceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// SaveWorkspaceConfig writes config back to the workspace.
func SaveWorkspaceConfig(dir string, cfg *WorkspaceConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

// --- Generators ---

func generateHeartbeatMD(keyName string) string {
	return fmt.Sprintf(`# PicoClaw Agent — Legible Money

## Tasks

### Chain Heartbeat
- **interval**: 3600
- **command**: exec ./skills/lgm-heartbeat-once.sh
- **description**: Send liveness heartbeat to Legible Money chain

### Verification Patrol
- **interval**: 30
- **command**: exec ./skills/lgm-check-rounds.sh
- **description**: Check for pending verification rounds

### Memory Sync
- **interval**: 1800
- **command**: exec ./skills/lgm-sync-home.sh
- **description**: Sync workspace memory to on-chain Agent Home

## Agent Info
- Key: %s
- Created: %s
`, keyName, time.Now().UTC().Format(time.RFC3339))
}

func generateEnvScript(keyName string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# LGM environment — auto-generated by picoclaw-plugin\n")
	sb.WriteString(fmt.Sprintf("export LGM_KEY=\"%s\"\n", keyName))
	sb.WriteString("export LGM_NODE=\"${LGM_NODE:-tcp://localhost:26657}\"\n")
	sb.WriteString("export LGM_CHAIN_ID=\"${LGM_CHAIN_ID:-legible-devnet-1}\"\n")
	sb.WriteString("export LGM_KEYRING=\"${LGM_KEYRING:-test}\"\n")
	sb.WriteString("export LGM_LEGBLED_HOME=\"${LGM_HOME:-$HOME/.legbled}\"\n")
	sb.WriteString("export LGM_SALT_DIR=\"${LGM_SALT_DIR:-$HOME/.lgm-agent/salts}\"\n")
	return sb.String()
}
