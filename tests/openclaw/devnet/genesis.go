//go:build openclaw

package devnet

import (
	"encoding/json"
	"fmt"
	"os"
)

// PatchGenesis patches genesis.json for OpenClaw test devnet.
// Go equivalent of cosmos-devnet.sh:146-185.
func PatchGenesis(genesisPath string) error {
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("reading genesis: %w", err)
	}

	var genesis map[string]interface{}
	if err := json.Unmarshal(data, &genesis); err != nil {
		return fmt.Errorf("parsing genesis: %w", err)
	}

	// Enable ABCI++ vote extensions at height 1
	setNestedValue(genesis, []string{"consensus", "params", "abci", "vote_extensions_enable_height"}, "1")

	appState, ok := genesis["app_state"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("genesis missing app_state")
	}

	// Staking: bond_denom = ulgm
	ensureNestedMap(appState, "staking", "params")
	appState["staking"].(map[string]interface{})["params"].(map[string]interface{})["bond_denom"] = "ulgm"

	// Knowledge: min_verifiers = 1 for single-validator mode
	ensureNestedMap(appState, "knowledge", "params")
	appState["knowledge"].(map[string]interface{})["params"].(map[string]interface{})["min_verifiers"] = 1

	// Governance: fast voting for tests
	ensureNestedMap(appState, "gov", "params")
	govParams := appState["gov"].(map[string]interface{})["params"].(map[string]interface{})
	govParams["voting_period"] = "60s"
	govParams["expedited_voting_period"] = "30s"
	govParams["max_deposit_period"] = "60s"
	govParams["min_deposit"] = []interface{}{
		map[string]interface{}{"denom": "ulgm", "amount": "10000000"},
	}

	// Bank: denom metadata
	ensureMap(appState, "bank")
	appState["bank"].(map[string]interface{})["denom_metadata"] = []interface{}{
		map[string]interface{}{
			"description": "The native staking token of Legible Money",
			"denom_units": []interface{}{
				map[string]interface{}{"denom": "ulgm", "exponent": 0, "aliases": []string{"microlegible"}},
				map[string]interface{}{"denom": "mlgm", "exponent": 3, "aliases": []string{"millilegible"}},
				map[string]interface{}{"denom": "lgm", "exponent": 6, "aliases": []string{"legible"}},
			},
			"base":    "ulgm",
			"display": "lgm",
			"name":    "Legible Money",
			"symbol":  "LGM",
		},
	}

	// Slashing: lenient for devnet
	ensureNestedMap(appState, "slashing", "params")
	slashingParams := appState["slashing"].(map[string]interface{})["params"].(map[string]interface{})
	slashingParams["signed_blocks_window"] = "100"
	slashingParams["slash_fraction_downtime"] = "0.010000000000000000"

	out, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling genesis: %w", err)
	}
	return os.WriteFile(genesisPath, out, 0644)
}

// setNestedValue sets a value at a nested map path, creating intermediate maps.
func setNestedValue(m map[string]interface{}, path []string, value interface{}) {
	for i, key := range path {
		if i == len(path)-1 {
			m[key] = value
			return
		}
		next, ok := m[key].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			m[key] = next
		}
		m = next
	}
}

// ensureMap ensures a key in the map holds a map[string]interface{}.
func ensureMap(m map[string]interface{}, key string) {
	if _, ok := m[key].(map[string]interface{}); !ok {
		m[key] = make(map[string]interface{})
	}
}

// ensureNestedMap ensures two levels of nested maps exist.
func ensureNestedMap(m map[string]interface{}, key1, key2 string) {
	ensureMap(m, key1)
	ensureMap(m[key1].(map[string]interface{}), key2)
}
