// tools/axiom-loader — Validates and injects genesis axioms into genesis.json.
//
// Usage:
//
//	go run tools/axiom-loader/main.go validate <axioms.json>
//	go run tools/axiom-loader/main.go inject  <axioms.json> <genesis.json>
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/legible-money/legible/x/knowledge/types"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: axiom-loader validate <axioms.json>\n")
			os.Exit(1)
		}
		cmdValidate(os.Args[2])

	case "inject":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: axiom-loader inject <axioms.json> <genesis.json>\n")
			os.Exit(1)
		}
		cmdInject(os.Args[2], os.Args[3])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader validate <axioms.json>              Validate axiom file\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader inject  <axioms.json> <genesis.json> Inject axioms into genesis\n")
}

// cmdValidate loads and validates an axiom file against the default domains.
func cmdValidate(axiomPath string) {
	axioms, err := types.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Build domain list from DefaultDomains
	domains := types.DefaultDomains()
	domainNames := make([]string, len(domains))
	for i, d := range domains {
		domainNames[i] = d.Name
	}

	if err := types.ValidateAxioms(axioms, domainNames); err != nil {
		fmt.Fprintf(os.Stderr, "VALIDATION FAILED: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	domainCounts := make(map[string]int)
	categoryCounts := make(map[string]int)
	typeCounts := make(map[string]int)
	for _, a := range axioms {
		domainCounts[a.Domain]++
		categoryCounts[a.EpistemicCategory]++
		if a.ClaimType != "" {
			typeCounts[a.ClaimType]++
		}
	}

	fmt.Printf("Axiom file validated successfully: %s\n", axiomPath)
	fmt.Printf("  Total axioms: %d\n", len(axioms))
	fmt.Printf("  Domains:\n")
	for domain, count := range domainCounts {
		fmt.Printf("    %-20s %d\n", domain, count)
	}
	fmt.Printf("  Categories:\n")
	for cat, count := range categoryCounts {
		fmt.Printf("    %-20s %d\n", cat, count)
	}
	if len(typeCounts) > 0 {
		fmt.Printf("  Claim Types:\n")
		for ct, count := range typeCounts {
			fmt.Printf("    %-20s %d\n", ct, count)
		}
	}
}

// cmdInject loads axioms, validates them, converts to Facts, and injects into genesis.json.
func cmdInject(axiomPath, genesisPath string) {
	// Load axioms
	axioms, err := types.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading axioms: %v\n", err)
		os.Exit(1)
	}

	// Load genesis
	genesisData, err := os.ReadFile(genesisPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR reading genesis: %v\n", err)
		os.Exit(1)
	}

	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(genesisData, &genesis); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing genesis: %v\n", err)
		os.Exit(1)
	}

	// Extract app_state
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genesis["app_state"], &appState); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing app_state: %v\n", err)
		os.Exit(1)
	}

	// Extract knowledge module state
	var knowledgeState map[string]json.RawMessage
	if err := json.Unmarshal(appState["knowledge"], &knowledgeState); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing knowledge state: %v\n", err)
		os.Exit(1)
	}

	// Get existing domains for validation
	var domains []*types.OntologicalDomain
	if domainsRaw, ok := knowledgeState["domains"]; ok {
		if err := json.Unmarshal(domainsRaw, &domains); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR parsing domains: %v\n", err)
			os.Exit(1)
		}
	}
	domainNames := make([]string, len(domains))
	for i, d := range domains {
		domainNames[i] = d.Name
	}

	// Validate axioms against genesis domains
	if err := types.ValidateAxioms(axioms, domainNames); err != nil {
		fmt.Fprintf(os.Stderr, "VALIDATION FAILED: %v\n", err)
		os.Exit(1)
	}

	// Convert to facts and fact references
	facts := types.AxiomsToFacts(axioms)
	factRefs := types.AxiomsToFactReferences(axioms)

	// Get existing facts (if any) and append
	var existingFacts []*types.Fact
	if factsRaw, ok := knowledgeState["facts"]; ok && string(factsRaw) != "null" {
		if err := json.Unmarshal(factsRaw, &existingFacts); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR parsing existing facts: %v\n", err)
			os.Exit(1)
		}
	}

	// Check for ID collisions
	existingIDs := make(map[string]bool, len(existingFacts))
	for _, f := range existingFacts {
		existingIDs[f.Id] = true
	}
	for _, f := range facts {
		if existingIDs[f.Id] {
			fmt.Fprintf(os.Stderr, "ERROR: axiom fact ID %q collides with existing fact\n", f.Id)
			os.Exit(1)
		}
	}

	allFacts := append(existingFacts, facts...)

	// Serialize facts back
	factsJSON, err := json.Marshal(allFacts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling facts: %v\n", err)
		os.Exit(1)
	}
	knowledgeState["facts"] = factsJSON

	// Inject FactReferences
	var existingRefs []*types.FactReference
	if refsRaw, ok := knowledgeState["fact_references"]; ok && string(refsRaw) != "null" {
		if err := json.Unmarshal(refsRaw, &existingRefs); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR parsing existing fact_references: %v\n", err)
			os.Exit(1)
		}
	}
	allRefs := append(existingRefs, factRefs...)
	refsJSON, err := json.Marshal(allRefs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling fact_references: %v\n", err)
		os.Exit(1)
	}
	knowledgeState["fact_references"] = refsJSON

	// Reassemble
	knowledgeJSON, err := json.Marshal(knowledgeState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling knowledge state: %v\n", err)
		os.Exit(1)
	}
	appState["knowledge"] = knowledgeJSON

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling app_state: %v\n", err)
		os.Exit(1)
	}
	genesis["app_state"] = appStateJSON

	// Write back with indentation
	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling genesis: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(genesisPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing genesis: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Injected %d axiom facts into %s\n", len(facts), genesisPath)
	fmt.Printf("  Total facts in genesis: %d\n", len(allFacts))

	// FactReference stats
	stats := types.ComputeFactReferenceStats(axioms, factRefs)
	fmt.Printf("Injected %d fact references into %s\n", len(factRefs), genesisPath)
	fmt.Printf("  Total references: %d\n", stats.Total)
	for refType, count := range stats.ByType {
		fmt.Printf("  Reference types: %s=%d\n", refType, count)
	}
	fmt.Printf("  Cross-domain references: %d\n", stats.CrossDomain)
	fmt.Printf("  Intra-domain references: %d\n", stats.IntraDomain)
}
