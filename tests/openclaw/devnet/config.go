//go:build openclaw

package devnet

import (
	"fmt"
	"os"
	"strings"
)

// PatchCometConfig patches CometBFT config.toml with dynamic ports
// and devnet-friendly settings (CORS, unsafe RPC).
func PatchCometConfig(configPath string, rpcPort, p2pPort, pprofPort int) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config.toml: %w", err)
	}
	content := string(data)

	// Replace default ports with dynamic ones
	content = strings.ReplaceAll(content, "tcp://0.0.0.0:26657", fmt.Sprintf("tcp://0.0.0.0:%d", rpcPort))
	content = strings.ReplaceAll(content, "tcp://0.0.0.0:26656", fmt.Sprintf("tcp://0.0.0.0:%d", p2pPort))
	content = strings.ReplaceAll(content, "localhost:6060", fmt.Sprintf("localhost:%d", pprofPort))

	// Enable CORS for devnet
	content = strings.ReplaceAll(content, `cors_allowed_origins = []`, `cors_allowed_origins = ["*"]`)

	// Enable unsafe RPC (first occurrence, in [rpc] section)
	content = strings.Replace(content, "unsafe = false", "unsafe = true", 1)

	return os.WriteFile(configPath, []byte(content), 0644)
}

// PatchAppConfig patches app.toml with dynamic ports and devnet-friendly settings.
func PatchAppConfig(appPath string, restPort, grpcPort, grpcWebPort int) error {
	data, err := os.ReadFile(appPath)
	if err != nil {
		return fmt.Errorf("reading app.toml: %w", err)
	}
	content := string(data)

	// Replace default ports with dynamic ones
	content = strings.ReplaceAll(content, "tcp://0.0.0.0:1317", fmt.Sprintf("tcp://0.0.0.0:%d", restPort))
	content = strings.ReplaceAll(content, "0.0.0.0:9090", fmt.Sprintf("0.0.0.0:%d", grpcPort))
	content = strings.ReplaceAll(content, "0.0.0.0:9091", fmt.Sprintf("0.0.0.0:%d", grpcWebPort))

	// Enable unsafe CORS (API is already enabled by default in initAppConfig)
	// NOTE: Do NOT flip gRPC enable — it panics without proto codegen.
	content = strings.ReplaceAll(content, "enabled-unsafe-cors = false", "enabled-unsafe-cors = true")

	return os.WriteFile(appPath, []byte(content), 0644)
}
