//go:build openclaw

package testnet

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/legible-money/legible/tests/openclaw/devnet"
)

// PatchGenesisMultiValidator applies the base devnet genesis patches
// plus multi-validator overrides (min_verifiers=2, etc).
func PatchGenesisMultiValidator(genesisPath string) error {
	// Apply base devnet patches first
	if err := devnet.PatchGenesis(genesisPath); err != nil {
		return err
	}

	// Re-read to apply multi-validator overrides
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("reading genesis for multi-val patches: %w", err)
	}

	var genesis map[string]interface{}
	if err := json.Unmarshal(data, &genesis); err != nil {
		return fmt.Errorf("parsing genesis for multi-val patches: %w", err)
	}

	appState, ok := genesis["app_state"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("genesis missing app_state")
	}

	// Knowledge: min_verifiers = 2 for multi-validator mode
	knowledgeParams := getNestedMap(appState, "knowledge", "params")
	if knowledgeParams != nil {
		knowledgeParams["min_verifiers"] = 2
	}

	// Emergency: min_distinct_voters = 2
	emergencyParams := getNestedMap(appState, "emergency", "params")
	if emergencyParams != nil {
		emergencyParams["min_distinct_voters"] = 2
	}

	out, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling multi-val genesis: %w", err)
	}
	return os.WriteFile(genesisPath, out, 0644)
}

// getNestedMap safely traverses two levels of nested maps.
// Returns nil if the path doesn't exist.
func getNestedMap(m map[string]interface{}, key1, key2 string) map[string]interface{} {
	v1, ok := m[key1].(map[string]interface{})
	if !ok {
		return nil
	}
	v2, ok := v1[key2].(map[string]interface{})
	if !ok {
		return nil
	}
	return v2
}
