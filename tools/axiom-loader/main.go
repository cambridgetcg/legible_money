// tools/axiom-loader — Validates, analyzes, and injects genesis axioms into genesis.json.
//
// Usage:
//
//	go run tools/axiom-loader/main.go validate <axioms.json>
//	go run tools/axiom-loader/main.go inject  <axioms.json> <genesis.json>
//	go run tools/axiom-loader/main.go stats   <axioms.json>
//	go run tools/axiom-loader/main.go edges   <axioms.json> [-o output.csv]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

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

	case "stats":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: axiom-loader stats <axioms.json>\n")
			os.Exit(1)
		}
		cmdStats(os.Args[2])

	case "edges":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: axiom-loader edges <axioms.json> [-o output.csv]\n")
			os.Exit(1)
		}
		outPath := ""
		for i := 3; i < len(os.Args)-1; i++ {
			if os.Args[i] == "-o" {
				outPath = os.Args[i+1]
				break
			}
		}
		cmdEdges(os.Args[2], outPath)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader validate <axioms.json>               Validate axiom file\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader inject  <axioms.json> <genesis.json>  Inject axioms into genesis\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader stats   <axioms.json>                 DAG analysis & domain matrix\n")
	fmt.Fprintf(os.Stderr, "  axiom-loader edges   <axioms.json> [-o out.csv]    Export dependency edges as CSV\n")
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

// cmdStats loads axioms and prints comprehensive DAG analysis.
func cmdStats(axiomPath string) {
	axioms, err := types.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Axiom DAG Analysis: %s ===\n\n", axiomPath)
	fmt.Printf("Total items: %d\n\n", len(axioms))

	// Items by type
	typeCounts := make(map[string]int)
	for _, a := range axioms {
		ct := a.ClaimType
		if ct == "" {
			ct = "(untyped)"
		}
		typeCounts[ct]++
	}
	fmt.Printf("--- Items by Type ---\n")
	typeKeys := sortedKeys(typeCounts)
	for _, ct := range typeKeys {
		fmt.Printf("  %-20s %d\n", ct, typeCounts[ct])
	}

	// Items by domain
	domainCounts := make(map[string]int)
	for _, a := range axioms {
		domainCounts[a.Domain]++
	}
	fmt.Printf("\n--- Items by Domain ---\n")
	domainKeys := sortedKeys(domainCounts)
	for _, d := range domainKeys {
		fmt.Printf("  %-20s %d\n", d, domainCounts[d])
	}

	// DAG stats
	dagStats, err := types.ComputeDAGStats(axioms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR computing DAG stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n--- DAG Topology ---\n")
	fmt.Printf("  Root count (no deps):  %d\n", dagStats.RootCount)
	fmt.Printf("  Max depth:             %d\n", dagStats.MaxDepth)
	fmt.Printf("  Total edges:           %d\n", dagStats.TotalEdges)
	fmt.Printf("  Orphan count:          %d\n", len(dagStats.Orphans))

	fmt.Printf("\n--- Depth Histogram ---\n")
	for d := 0; d <= dagStats.MaxDepth; d++ {
		count := dagStats.DepthCounts[d]
		bar := strings.Repeat("#", count)
		fmt.Printf("  depth %2d: %3d  %s\n", d, count, bar)
	}

	// Top most-referenced
	if len(dagStats.MostReferenced) > 0 {
		fmt.Printf("\n--- Top %d Most-Referenced (Hub Nodes) ---\n", len(dagStats.MostReferenced))
		for i, entry := range dagStats.MostReferenced {
			fmt.Printf("  %2d. %s\n", i+1, entry)
		}
	}

	// Cross-domain matrix
	matrix := types.ComputeCrossDomainMatrix(axioms)
	if len(matrix.Entries) > 0 {
		fmt.Printf("\n--- Cross-Domain Dependency Matrix ---\n")
		fmt.Printf("  %-20s → %-20s  Count\n", "From", "To")
		fmt.Printf("  %s\n", strings.Repeat("-", 50))
		for _, e := range matrix.Entries {
			fmt.Printf("  %-20s → %-20s  %d\n", e.FromDomain, e.ToDomain, e.Count)
		}
	}

	// Warnings
	warnings := 0
	if len(dagStats.Orphans) > 0 {
		if warnings == 0 {
			fmt.Printf("\n--- Warnings ---\n")
		}
		warnings++
		fmt.Printf("  [WARN] %d orphan(s) (no deps, nothing depends on them):\n", len(dagStats.Orphans))
		for _, o := range dagStats.Orphans {
			fmt.Printf("         %s\n", o)
		}
	}
	if dagStats.MaxDepth > 10 {
		if warnings == 0 {
			fmt.Printf("\n--- Warnings ---\n")
		}
		warnings++
		highDepth := 0
		for d := 11; d <= dagStats.MaxDepth; d++ {
			highDepth += dagStats.DepthCounts[d]
		}
		fmt.Printf("  [WARN] %d item(s) at depth > 10 (max depth: %d)\n", highDepth, dagStats.MaxDepth)
	}

	fmt.Println()
}

// cmdEdges exports all dependency edges as CSV.
func cmdEdges(axiomPath string, outPath string) {
	axioms, err := types.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Build domain lookup
	idToDomain := make(map[string]string, len(axioms))
	for _, a := range axioms {
		idToDomain[a.AxiomID] = a.Domain
	}

	var lines []string
	lines = append(lines, "from,to,from_domain,to_domain")
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			depDomain := idToDomain[dep]
			if depDomain == "" {
				if dashIdx := strings.Index(dep, "-"); dashIdx > 0 {
					if d, ok := types.PrefixToDomainMap[dep[:dashIdx]]; ok {
						depDomain = d
					}
				}
			}
			lines = append(lines, fmt.Sprintf("%s,%s,%s,%s", a.AxiomID, dep, a.Domain, depDomain))
		}
	}

	output := strings.Join(lines, "\n") + "\n"

	if outPath != "" {
		if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR writing %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %d edges to %s\n", len(lines)-1, outPath)
	} else {
		fmt.Print(output)
	}
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
