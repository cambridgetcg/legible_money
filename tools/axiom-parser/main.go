// tools/axiom-parser — Parses structured markdown axiom files into JSON.
//
// Reads all domain .md files from genesis-axioms/ and emits a single JSON array
// of GenesisAxiom structs compatible with tools/axiom-loader.
//
// Usage:
//
//	go run tools/axiom-parser/main.go -dir genesis-axioms/ -o scripts/axioms-777.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/legible-money/legible/x/knowledge/types"
)

// headerRegex matches the axiom header line:
// PREFIX-NNN[suffix] (Type, Title)
// Examples: MATH-001 (Axiom, Domain Nonempty), AGRT-004a-i (Derived, Forking Doctrine)
var headerRegex = regexp.MustCompile(`^([A-Z]+-\d{3}[a-z0-9\-]*)\s+\((.+)\)$`)

// knownFields are the field names recognized in key: value lines within entries.
var knownFields = map[string]bool{
	"type": true, "domain": true, "confidence": true,
	"depends": true, "formal": true, "formal_expression": true,
	"regime": true, "approximation_tags": true,
	"source_tradition": true, "source": true,
	"evidence_ref": true, "derivation_ref": true,
	"notes": true, "note": true,
	"references": true, "tier_required": true,
}

// headerTypeMap normalizes header type strings to GenesisAxiom ClaimType values.
var headerTypeMap = map[string]string{
	"axiom":              "axiom",
	"axiom schema":       "axiom",
	"definition":         "definition",
	"empirical axiom":    "empirical_axiom",
	"empirical":          "empirical_axiom",
	"derived":            "derived_claim",
	"derived claim":      "derived_claim",
	"measurement fact":   "measurement_fact",
	"measurement":        "measurement_fact",
	"meta":               "meta",
	"regime declaration": "regime_declaration",
	"regime":             "regime_declaration",
	"verified":           "derived_claim",
	"meta-verified":      "meta",
}

// epistemicMap maps ClaimType to EpistemicCategory.
var epistemicMap = map[string]string{
	"axiom":              "analytic",
	"definition":         "analytic",
	"empirical_axiom":    "empirical",
	"regime_declaration": "formal",
	"derived_claim":      "formal",
	"measurement_fact":   "empirical",
	"meta":               "empirical",
}

func main() {
	dir := flag.String("dir", "genesis-axioms", "directory containing .md axiom files")
	output := flag.String("o", "", "output file (default: stdout)")
	flag.Parse()

	axioms, err := ParseDirectory(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Clamp derived claim confidence to not exceed weakest dependency.
	clamped := clampDerivedConfidence(axioms)
	if clamped > 0 {
		fmt.Fprintf(os.Stderr, "Clamped %d derived claims to match dependency confidence\n", clamped)
	}

	// Sort by axiom_id for deterministic output.
	sort.Slice(axioms, func(i, j int) bool {
		return axioms[i].AxiomID < axioms[j].AxiomID
	})

	data, err := json.MarshalIndent(axioms, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d axioms to %s\n", len(axioms), *output)
	} else {
		fmt.Println(string(data))
	}
}

// ParseDirectory reads all domain .md files from a directory and parses axioms.
// Files are identified by name matching a known prefix (MATH, PHYS, etc.).
func ParseDirectory(dir string) ([]*types.GenesisAxiom, error) {
	var all []*types.GenesisAxiom

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		// Only process files whose name matches a known domain prefix.
		name := strings.TrimSuffix(d.Name(), ".md")
		if _, ok := types.PrefixToDomainMap[name]; !ok {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		axioms, err := ParseFile(string(data))
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		all = append(all, axioms...)
		return nil
	})

	return all, err
}

// ParseFile parses all axiom entries from a single markdown file's code blocks.
func ParseFile(content string) ([]*types.GenesisAxiom, error) {
	var axioms []*types.GenesisAxiom

	blocks := extractCodeBlocks(content)
	for _, block := range blocks {
		entries := splitEntries(block)
		for _, entry := range entries {
			axiom, err := ParseEntry(entry)
			if err != nil {
				return nil, err
			}
			if axiom != nil {
				axioms = append(axioms, axiom)
			}
		}
	}

	return axioms, nil
}

// extractCodeBlocks finds all content between triple-backtick fences.
func extractCodeBlocks(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")
	var inBlock bool
	var block []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				blocks = append(blocks, strings.Join(block, "\n"))
				block = nil
				inBlock = false
			} else {
				inBlock = true
			}
			continue
		}
		if inBlock {
			block = append(block, line)
		}
	}

	return blocks
}

// splitEntries splits a code block into individual axiom entries
// separated by blank lines.
func splitEntries(block string) [][]string {
	lines := strings.Split(block, "\n")
	var entries [][]string
	var current []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				entries = append(entries, current)
				current = nil
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		entries = append(entries, current)
	}

	return entries
}

// ParseEntry parses a single axiom entry (header + body lines) into a GenesisAxiom.
// Returns nil for non-axiom content (lines that don't start with a valid header).
func ParseEntry(lines []string) (*types.GenesisAxiom, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	// Parse header: PREFIX-NNN[suffix] (Type, Title)
	header := strings.TrimSpace(lines[0])
	match := headerRegex.FindStringSubmatch(header)
	if match == nil {
		return nil, nil
	}

	axiomID := match[1]
	headerInfo := match[2]

	// Extract the header type and title.
	// Format: "Type, Title" — e.g., "Axiom, Domain Nonempty"
	headerType := headerInfo
	headerTitle := ""
	if idx := strings.Index(headerInfo, ","); idx >= 0 {
		headerType = strings.TrimSpace(headerInfo[:idx])
		headerTitle = strings.TrimSpace(headerInfo[idx+1:])
	}

	// Parse body into statement lines and key:value fields.
	var statementLines []string
	fields := make(map[string]string)
	var currentKey string
	inStatement := true

	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		fieldKey, fieldVal, isField := parseFieldLine(trimmed)

		if isField && inStatement {
			inStatement = false
		}

		if isField {
			currentKey = fieldKey
			fields[currentKey] = fieldVal
		} else if inStatement {
			statementLines = append(statementLines, trimmed)
		} else if currentKey != "" {
			// Continuation line for the current field value.
			fields[currentKey] += " " + trimmed
		}
	}

	// Build statement: use parsed text, or fall back to header title
	// (some entries like "MATH-060 (Verified, |Sₙ|=n!)" have no body text).
	statement := strings.Join(statementLines, " ")
	if statement == "" && headerTitle != "" {
		statement = headerTitle
	}

	axiom := &types.GenesisAxiom{
		AxiomID:   axiomID,
		Statement: statement,
	}

	// --- Map explicit fields ---

	// ClaimType: explicit type: field, or inferred from header.
	// Compound annotations like "measurement_fact (asymmetry) + derived_claim (conditions)"
	// are normalized to the first known type ("measurement_fact").
	if v, ok := fields["type"]; ok {
		axiom.ClaimType = normalizeClaimType(strings.TrimSpace(v))
	} else {
		axiom.ClaimType = inferType(headerType)
	}

	// Domain: explicit domain: field, or inferred from ID prefix.
	if v, ok := fields["domain"]; ok {
		axiom.Domain = strings.TrimSpace(v)
	} else {
		prefix := extractPrefix(axiomID)
		if d, ok := types.PrefixToDomainMap[prefix]; ok {
			axiom.Domain = d
		}
	}

	// Confidence: explicit, or default from claim type.
	if v, ok := fields["confidence"]; ok {
		conf, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("axiom %s: invalid confidence %q: %w", axiomID, v, err)
		}
		axiom.Confidence = conf
	} else {
		if def, ok := types.ClaimTypeDefaultConfidence[axiom.ClaimType]; ok {
			axiom.Confidence = def
		} else {
			axiom.Confidence = 1.0
		}
	}

	// Formal expression (formal: or formal_expression:).
	if v, ok := fields["formal"]; ok {
		axiom.FormalExpression = strings.TrimSpace(v)
	}
	if v, ok := fields["formal_expression"]; ok {
		axiom.FormalExpression = strings.TrimSpace(v)
	}

	// Regime.
	if v, ok := fields["regime"]; ok {
		axiom.Regime = strings.TrimSpace(v)
	}

	// Approximation tags.
	if v, ok := fields["approximation_tags"]; ok {
		axiom.ApproxTags = parseList(v)
	}

	// Dependencies (always non-nil for JSON serialization).
	if v, ok := fields["depends"]; ok {
		axiom.Dependencies = parseList(v)
	}
	if axiom.Dependencies == nil {
		axiom.Dependencies = []string{}
	}

	// Source tradition (source_tradition: takes priority over source:).
	if v, ok := fields["source_tradition"]; ok {
		axiom.SourceTradition = stripQuotes(strings.TrimSpace(v))
	} else if v, ok := fields["source"]; ok {
		axiom.SourceTradition = stripQuotes(strings.TrimSpace(v))
	}

	// Notes (notes: takes priority over note:).
	if v, ok := fields["notes"]; ok {
		axiom.Notes = strings.TrimSpace(v)
	} else if v, ok := fields["note"]; ok {
		axiom.Notes = strings.TrimSpace(v)
	}

	// Evidence ref.
	if v, ok := fields["evidence_ref"]; ok {
		axiom.EvidenceRef = strings.TrimSpace(v)
	}

	// Derivation ref.
	if v, ok := fields["derivation_ref"]; ok {
		axiom.DerivationRef = strings.TrimSpace(v)
	}

	// References.
	if v, ok := fields["references"]; ok {
		axiom.References = parseList(v)
	}

	// Tier required.
	if v, ok := fields["tier_required"]; ok {
		tier, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("axiom %s: invalid tier_required %q: %w", axiomID, v, err)
		}
		axiom.TierRequired = tier
	}

	// Auto-map epistemic category from claim type.
	if cat, ok := epistemicMap[axiom.ClaimType]; ok {
		axiom.EpistemicCategory = cat
	} else {
		axiom.EpistemicCategory = "empirical"
	}

	return axiom, nil
}

// parseFieldLine checks if a trimmed line matches "key: value" for a known field.
func parseFieldLine(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	if !knownFields[key] {
		return "", "", false
	}
	value = strings.TrimSpace(line[idx+1:])
	return key, value, true
}

// parseList parses bracket-delimited lists like "[item1, item2, ...]" into a string slice.
func parseList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}

// stripQuotes removes surrounding double quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// extractPrefix returns the letter prefix from an axiom ID (e.g., "MATH" from "MATH-001").
func extractPrefix(id string) string {
	if idx := strings.Index(id, "-"); idx >= 0 {
		return id[:idx]
	}
	return id
}

// inferType maps a header type string (e.g., "Axiom Schema") to a ClaimType value.
func inferType(headerType string) string {
	normalized := strings.ToLower(strings.TrimSpace(headerType))
	if mapped, ok := headerTypeMap[normalized]; ok {
		return mapped
	}
	return ""
}

// knownClaimTypes are the valid claim types recognized by the knowledge module.
var knownClaimTypes = []string{
	"axiom", "definition", "empirical_axiom", "regime_declaration",
	"derived_claim", "measurement_fact", "meta",
}

// normalizeClaimType extracts the first valid claim type from a possibly
// compound annotation like "measurement_fact (asymmetry) + derived_claim (conditions)".
func normalizeClaimType(raw string) string {
	for _, ct := range knownClaimTypes {
		if strings.HasPrefix(raw, ct) {
			return ct
		}
	}
	return raw
}

// clampDerivedConfidence ensures derived claims don't have higher confidence
// than their weakest dependency. Returns the number of axioms clamped.
func clampDerivedConfidence(axioms []*types.GenesisAxiom) int {
	confOf := make(map[string]float64, len(axioms))
	for _, a := range axioms {
		conf := a.Confidence
		if conf == 0 {
			if def, ok := types.ClaimTypeDefaultConfidence[a.ClaimType]; ok {
				conf = def
			} else {
				conf = 1.0
			}
		}
		confOf[a.AxiomID] = conf
	}

	clamped := 0
	for _, a := range axioms {
		if a.ClaimType != "derived_claim" || len(a.Dependencies) == 0 {
			continue
		}
		minDep := 2.0
		for _, dep := range a.Dependencies {
			if c, ok := confOf[dep]; ok && c < minDep {
				minDep = c
			}
		}
		if minDep < 2.0 && confOf[a.AxiomID] > minDep {
			a.Confidence = minDep
			confOf[a.AxiomID] = minDep
			clamped++
		}
	}
	return clamped
}
