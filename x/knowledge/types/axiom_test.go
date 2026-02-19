package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAxiomID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"MATH-001", false},
		{"THEO-042", false},
		{"LOGIC-999", false},
		{"CS-001", false},
		{"ETHIC-001", false},
		{"AGRT-001", false},
		{"AGRT-091", false},
		{"INFO-123", false},
		{"AGRT-004a", false},     // lowercase suffix allowed
		{"AGRT-004a-ii", false},  // multi-part suffix allowed
		// Invalid
		{"ETHICS-001", true}, // ETHICS prefix no longer valid (now ETHIC)
		{"MATH001", true},    // no dash
		{"math-001", true},   // lowercase
		{"MATH-01", true},    // only 2 digits
		{"MATH-", true},      // no digits
		{"UNKNOWN-001", true}, // unknown prefix
		{"", true},
	}
	for _, tt := range tests {
		err := ValidateAxiomID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateAxiomID(%q): got err=%v, wantErr=%v", tt.id, err, tt.wantErr)
		}
	}
}

func TestDomainPrefixMapConsistency(t *testing.T) {
	// Every domain in DomainPrefixMap should reverse-map correctly
	for domain, prefix := range DomainPrefixMap {
		got, ok := PrefixToDomainMap[prefix]
		if !ok {
			t.Errorf("prefix %q not in PrefixToDomainMap", prefix)
		}
		if got != domain {
			t.Errorf("PrefixToDomainMap[%q] = %q, want %q", prefix, got, domain)
		}
	}
	// Exactly 15 domains
	if len(DomainPrefixMap) != 16 {
		t.Errorf("DomainPrefixMap has %d entries, want 16", len(DomainPrefixMap))
	}
}

func TestParseAxioms(t *testing.T) {
	data := `[
		{
			"axiom_id": "MATH-001",
			"statement": "Zero is a natural number",
			"formal_expression": "0 in N",
			"domain": "mathematics",
			"epistemic_category": "analytic",
			"references": [],
			"dependencies": []
		},
		{
			"axiom_id": "MATH-002",
			"statement": "Successor closure",
			"formal_expression": "",
			"domain": "mathematics",
			"epistemic_category": "analytic",
			"references": ["MATH-001"],
			"dependencies": ["MATH-001"]
		}
	]`
	axioms, err := ParseAxioms([]byte(data))
	if err != nil {
		t.Fatalf("ParseAxioms: %v", err)
	}
	if len(axioms) != 2 {
		t.Fatalf("got %d axioms, want 2", len(axioms))
	}
	if axioms[0].AxiomID != "MATH-001" {
		t.Errorf("axiom[0].AxiomID = %q, want MATH-001", axioms[0].AxiomID)
	}
	if axioms[1].Dependencies[0] != "MATH-001" {
		t.Errorf("axiom[1].Dependencies[0] = %q, want MATH-001", axioms[1].Dependencies[0])
	}
}

func TestValidateAxioms_Valid(t *testing.T) {
	axioms := []*GenesisAxiom{
		{
			AxiomID:          "MATH-001",
			Statement:        "Zero is a natural number",
			Domain:           "mathematics",
			EpistemicCategory: "analytic",
		},
		{
			AxiomID:          "MATH-002",
			Statement:        "Successor closure",
			Domain:           "mathematics",
			EpistemicCategory: "formal",
			References:       []string{"MATH-001"},
			Dependencies:     []string{"MATH-001"},
		},
		{
			AxiomID:          "LOGIC-001",
			Statement:        "Law of non-contradiction",
			Domain:           "logic",
			EpistemicCategory: "analytic",
		},
	}
	domains := []string{"mathematics", "logic", "physics"}
	if err := ValidateAxioms(axioms, domains); err != nil {
		t.Errorf("ValidateAxioms: unexpected error: %v", err)
	}
}

func TestValidateAxioms_DuplicateID(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-001", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic"},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected duplicate ID error")
	}
}

func TestValidateAxioms_UnknownDomain(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "alchemy", EpistemicCategory: "analytic"},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected unknown domain error")
	}
}

func TestValidateAxioms_InvalidCategory(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "magical"},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected invalid category error")
	}
}

func TestValidateAxioms_EmpiricalCategoryNowValid(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "PHYS-001", Statement: "Energy is conserved", ClaimType: "empirical_axiom",
			Domain: "physics", EpistemicCategory: "empirical", Confidence: 0.99},
	}
	err := ValidateAxioms(axioms, []string{"physics"})
	if err != nil {
		t.Errorf("empirical category should be valid: %v", err)
	}
}

func TestValidateAxioms_ProtocolCategoryValid(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "PHYS-001", Statement: "Protocol observation", ClaimType: "axiom",
			Domain: "physics", EpistemicCategory: "protocol", Confidence: 1.0},
	}
	err := ValidateAxioms(axioms, []string{"physics"})
	if err != nil {
		t.Errorf("protocol category should be valid: %v", err)
	}
}

func TestValidateAxioms_InvalidClaimType(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", ClaimType: "hypothesis",
			Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected invalid claim type error")
	}
}

func TestValidateAxioms_InvalidRegime(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "PHYS-001", Statement: "A", ClaimType: "empirical_axiom",
			Domain: "physics", EpistemicCategory: "empirical", Confidence: 0.99, Regime: "alchemy"},
	}
	err := ValidateAxioms(axioms, []string{"physics"})
	if err == nil {
		t.Error("expected invalid regime error")
	}
}

func TestValidateAxioms_ConfidenceOutOfRange(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics",
			EpistemicCategory: "analytic", Confidence: 1.5},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected confidence out of range error")
	}
}

func TestValidateAxioms_NegativeConfidence(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics",
			EpistemicCategory: "analytic", Confidence: -0.1},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected negative confidence error")
	}
}

func TestValidateAxioms_DerivedClaimNeedsDependencies(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", ClaimType: "derived_claim",
			Domain: "mathematics", EpistemicCategory: "formal", Confidence: 1.0},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected derived_claim without dependencies error")
	}
}

func TestValidateAxioms_DerivedClaimWithDependencies(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "Base axiom", ClaimType: "axiom",
			Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
		{AxiomID: "MATH-002", Statement: "Derived from base", ClaimType: "derived_claim",
			Domain: "mathematics", EpistemicCategory: "formal", Confidence: 1.0,
			Dependencies: []string{"MATH-001"}},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err != nil {
		t.Errorf("derived_claim with dependencies should be valid: %v", err)
	}
}

func TestValidateAxioms_ValidRegime(t *testing.T) {
	for _, regime := range []string{"classical", "SR", "GR", "QM", "QFT", "thermo", "statmech", "cosmology", "SM"} {
		axioms := []*GenesisAxiom{
			{AxiomID: "PHYS-001", Statement: "A", ClaimType: "empirical_axiom",
				Domain: "physics", EpistemicCategory: "empirical", Confidence: 0.99, Regime: regime},
		}
		if err := ValidateAxioms(axioms, []string{"physics"}); err != nil {
			t.Errorf("regime %q should be valid: %v", regime, err)
		}
	}
}

func TestValidClaimTypesCompleteness(t *testing.T) {
	expected := []string{"axiom", "empirical_axiom", "definition", "regime_declaration", "derived_claim", "measurement_fact", "meta"}
	if len(ValidClaimTypes) != len(expected) {
		t.Errorf("ValidClaimTypes has %d entries, want %d", len(ValidClaimTypes), len(expected))
	}
	for _, ct := range expected {
		if !ValidClaimTypes[ct] {
			t.Errorf("ValidClaimTypes missing %q", ct)
		}
	}
}

func TestValidateAxioms_PrefixDomainMismatch(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "physics", EpistemicCategory: "analytic"},
	}
	err := ValidateAxioms(axioms, []string{"mathematics", "physics"})
	if err == nil {
		t.Error("expected prefix-domain mismatch error")
	}
}

func TestValidateAxioms_EmptyStatement(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "", Domain: "mathematics", EpistemicCategory: "analytic"},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected empty statement error")
	}
}

func TestValidateAxioms_UnknownReference(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic",
			References: []string{"MATH-999"}},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected unknown reference error")
	}
}

func TestValidateAxiomDAG_NoCycle(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-003", Statement: "C", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001", "MATH-002"}},
	}
	if err := ValidateAxiomDAG(axioms); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAxiomDAG_CycleDetection(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-003"}},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-003", Statement: "C", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-002"}},
	}
	err := ValidateAxiomDAG(axioms)
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestValidateAxiomDAG_SelfDependency(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
	}
	err := ValidateAxiomDAG(axioms)
	if err == nil {
		t.Error("expected self-dependency error")
	}
}

func TestValidateAxiomDAG_MissingDependency(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-999"}},
	}
	err := ValidateAxiomDAG(axioms)
	if err == nil {
		t.Error("expected missing dependency error")
	}
}

func TestAxiomsToFacts(t *testing.T) {
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "MATH-001",
			Statement:         "Zero is a natural number",
			FormalExpression:   "0 in N",
			ClaimType:         "axiom",
			Domain:            "mathematics",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
		},
		{
			AxiomID:           "LOGIC-001",
			Statement:         "Law of non-contradiction",
			ClaimType:         "axiom",
			Domain:            "logic",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{"MATH-001"},
		},
	}

	facts := AxiomsToFacts(axioms)
	if len(facts) != 2 {
		t.Fatalf("got %d facts, want 2", len(facts))
	}

	// Check MATH-001
	f := facts[0]
	if f.Id != "MATH-001" {
		t.Errorf("fact[0].Id = %q, want MATH-001", f.Id)
	}
	if f.ClaimText != "Zero is a natural number" {
		t.Errorf("fact[0].ClaimText = %q", f.ClaimText)
	}
	if f.Domain != "mathematics" {
		t.Errorf("fact[0].Domain = %q", f.Domain)
	}
	if f.EpistemicCategory != "analytic" {
		t.Errorf("fact[0].EpistemicCategory = %q", f.EpistemicCategory)
	}
	if f.Confidence != AxiomConfidence {
		t.Errorf("fact[0].Confidence = %d, want %d", f.Confidence, AxiomConfidence)
	}
	if f.BaseConfidence != AxiomConfidence {
		t.Errorf("fact[0].BaseConfidence = %d, want %d", f.BaseConfidence, AxiomConfidence)
	}
	if f.Status != FactStatusVerified {
		t.Errorf("fact[0].Status = %q, want %q", f.Status, FactStatusVerified)
	}
	if f.Submitter != AxiomSubmitter {
		t.Errorf("fact[0].Submitter = %q, want %q", f.Submitter, AxiomSubmitter)
	}
	if f.SubmitterDid != AxiomSubmitterDID {
		t.Errorf("fact[0].SubmitterDid = %q, want %q", f.SubmitterDid, AxiomSubmitterDID)
	}
	if f.Maturity != MaturityCanonical {
		t.Errorf("fact[0].Maturity = %q, want %q", f.Maturity, MaturityCanonical)
	}
	if f.IsProvisional {
		t.Error("fact[0].IsProvisional should be false")
	}
	if f.VerifiedAtHeight != 0 {
		t.Errorf("fact[0].VerifiedAtHeight = %d, want 0", f.VerifiedAtHeight)
	}

	// Check LOGIC-001 has dependencies as references
	f2 := facts[1]
	if len(f2.References) != 1 || f2.References[0] != "MATH-001" {
		t.Errorf("fact[1].References = %v, want [MATH-001]", f2.References)
	}
}

func TestAxiomsToFacts_PerAxiomConfidence(t *testing.T) {
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "PHYS-001",
			Statement:         "Conservation of energy",
			ClaimType:         "empirical_axiom",
			Domain:            "physics",
			EpistemicCategory: "empirical",
			Confidence:        0.99,
		},
		{
			AxiomID:           "COSM-001",
			Statement:         "Universe is expanding",
			ClaimType:         "measurement_fact",
			Domain:            "cosmology",
			EpistemicCategory: "empirical",
			Confidence:        0.95,
		},
	}

	facts := AxiomsToFacts(axioms)
	if len(facts) != 2 {
		t.Fatalf("got %d facts, want 2", len(facts))
	}

	// PHYS-001: 0.99 * 1_000_000 = 990_000
	if facts[0].Confidence != 990_000 {
		t.Errorf("PHYS-001 confidence = %d, want 990000", facts[0].Confidence)
	}
	if facts[0].BaseConfidence != 990_000 {
		t.Errorf("PHYS-001 base_confidence = %d, want 990000", facts[0].BaseConfidence)
	}

	// COSM-001: 0.95 * 1_000_000 = 950_000
	if facts[1].Confidence != 950_000 {
		t.Errorf("COSM-001 confidence = %d, want 950000", facts[1].Confidence)
	}
}

func TestAxiomsToFacts_DefaultConfidenceFallback(t *testing.T) {
	// When Confidence is 0 and ClaimType is set, use default for that type
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "MATH-001",
			Statement:         "Zero is natural",
			ClaimType:         "axiom",
			Domain:            "mathematics",
			EpistemicCategory: "analytic",
			Confidence:        0, // should default to 1.0 for axiom type
		},
	}
	facts := AxiomsToFacts(axioms)
	if facts[0].Confidence != 1_000_000 {
		t.Errorf("default axiom confidence = %d, want 1000000", facts[0].Confidence)
	}
}

func TestAxiomsToFacts_LegacyNoClaimType(t *testing.T) {
	// Legacy axiom without ClaimType defaults to 1.0 confidence
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "MATH-001",
			Statement:         "Zero is natural",
			Domain:            "mathematics",
			EpistemicCategory: "analytic",
			// No ClaimType, no Confidence
		},
	}
	facts := AxiomsToFacts(axioms)
	if facts[0].Confidence != 1_000_000 {
		t.Errorf("legacy axiom confidence = %d, want 1000000", facts[0].Confidence)
	}
}

func TestAxiomsToFacts_RoundTrip(t *testing.T) {
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "PHYS-001",
			Statement:         "Conservation of energy",
			ClaimType:         "empirical_axiom",
			Domain:            "physics",
			EpistemicCategory: "empirical",
			Confidence:        0.99,
			Regime:            "classical",
		},
	}
	facts := AxiomsToFacts(axioms)

	// Marshal and unmarshal to verify JSON round-trip
	bz, err := json.Marshal(facts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed []*Fact
	if err := json.Unmarshal(bz, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("got %d facts after round-trip, want 1", len(parsed))
	}
	if parsed[0].Id != "PHYS-001" {
		t.Errorf("round-trip Id = %q, want PHYS-001", parsed[0].Id)
	}
}

func TestLoadAxiomsFromFile(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "test-axioms.json")

	data := `[
		{
			"axiom_id": "MATH-001",
			"statement": "Zero is a natural number",
			"formal_expression": "0 in N",
			"domain": "mathematics",
			"epistemic_category": "analytic",
			"references": [],
			"dependencies": []
		}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	axioms, err := LoadAxiomsFromFile(path)
	if err != nil {
		t.Fatalf("LoadAxiomsFromFile: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("got %d axioms, want 1", len(axioms))
	}
	if axioms[0].AxiomID != "MATH-001" {
		t.Errorf("axiom ID = %q, want MATH-001", axioms[0].AxiomID)
	}
}

func TestLoadAxiomsFromFile_NotFound(t *testing.T) {
	_, err := LoadAxiomsFromFile("/nonexistent/file.json")
	if err == nil {
		t.Error("expected file not found error")
	}
}

func TestValidateAxioms_Empty(t *testing.T) {
	err := ValidateAxioms([]*GenesisAxiom{}, []string{"mathematics"})
	if err == nil {
		t.Error("expected empty axiom set error")
	}
}

func TestValidateAxioms_NilAxiom(t *testing.T) {
	axioms := []*GenesisAxiom{nil}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected nil axiom error")
	}
}

func TestGenesisValidate_FactsWithDomains(t *testing.T) {
	gs := &GenesisState{
		Params:  nil,
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
			{Id: "MATH-002", Domain: "mathematics", EpistemicCategory: "formal", References: []string{"MATH-001"}},
		},
	}
	if err := gs.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenesisValidate_DuplicateFactID(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected duplicate fact ID error")
	}
}

func TestGenesisValidate_UnknownFactDomain(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "TEST-001", Domain: "alchemy", EpistemicCategory: "analytic"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected unknown domain error")
	}
}

func TestGenesisValidate_UnknownFactCategory(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "TEST-001", Domain: "mathematics", EpistemicCategory: "magical"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected unknown category error")
	}
}

func TestGenesisValidate_InvalidFactReference(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic", References: []string{"NONEXISTENT"}},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected invalid reference error")
	}
}

func TestDefaultDomainsCount(t *testing.T) {
	domains := DefaultDomains()
	if len(domains) != 17 {
		t.Errorf("DefaultDomains() has %d domains, want 17", len(domains))
	}

	// Verify all domains have names and active status
	for _, d := range domains {
		if d.Name == "" {
			t.Error("domain has empty name")
		}
		if d.Status != "active" {
			t.Errorf("domain %q has status %q, want active", d.Name, d.Status)
		}
	}
}

func TestAxiomDomainNames(t *testing.T) {
	names := AxiomDomainNames()
	if len(names) != 16 {
		t.Errorf("AxiomDomainNames() has %d entries, want 16", len(names))
	}
}

func TestClaimTypeToCategory(t *testing.T) {
	// All 7 claim types must map to a valid epistemic category
	expected := map[string]string{
		"axiom":              "analytic",
		"definition":         "analytic",
		"regime_declaration": "formal",
		"derived_claim":      "formal",
		"empirical_axiom":    "empirical",
		"measurement_fact":   "empirical",
		"meta":               "empirical",
	}
	if len(ClaimTypeToCategory) != len(expected) {
		t.Errorf("ClaimTypeToCategory has %d entries, want %d", len(ClaimTypeToCategory), len(expected))
	}
	for ct, wantCat := range expected {
		got, ok := ClaimTypeToCategory[ct]
		if !ok {
			t.Errorf("ClaimTypeToCategory missing %q", ct)
			continue
		}
		if got != wantCat {
			t.Errorf("ClaimTypeToCategory[%q] = %q, want %q", ct, got, wantCat)
		}
		// All mapped categories must be in ValidAxiomCategories
		if !ValidAxiomCategories[got] {
			t.Errorf("ClaimTypeToCategory[%q] maps to %q which is not a valid category", ct, got)
		}
	}
}

func TestAxiomsToFactsCategory_AutoDerive(t *testing.T) {
	// When EpistemicCategory is empty, category is auto-derived from ClaimType
	axioms := []*GenesisAxiom{
		{
			AxiomID:    "MATH-001",
			Statement:  "Zero is a natural number",
			ClaimType:  "axiom",
			Domain:     "mathematics",
			Confidence: 1.0,
		},
		{
			AxiomID:    "PHYS-001",
			Statement:  "Energy is conserved",
			ClaimType:  "empirical_axiom",
			Domain:     "physics",
			Confidence: 0.99,
		},
		{
			AxiomID:      "MATH-002",
			Statement:    "Derived from zero",
			ClaimType:    "derived_claim",
			Domain:       "mathematics",
			Confidence:   0.95,
			Dependencies: []string{"MATH-001"},
		},
	}

	facts := AxiomsToFacts(axioms)
	if len(facts) != 3 {
		t.Fatalf("got %d facts, want 3", len(facts))
	}

	// axiom → analytic
	if facts[0].EpistemicCategory != "analytic" {
		t.Errorf("axiom type → category = %q, want analytic", facts[0].EpistemicCategory)
	}
	// empirical_axiom → empirical
	if facts[1].EpistemicCategory != "empirical" {
		t.Errorf("empirical_axiom type → category = %q, want empirical", facts[1].EpistemicCategory)
	}
	// derived_claim → formal
	if facts[2].EpistemicCategory != "formal" {
		t.Errorf("derived_claim type → category = %q, want formal", facts[2].EpistemicCategory)
	}
}

func TestAxiomsToFactsCategory_ExplicitOverride(t *testing.T) {
	// When EpistemicCategory is explicitly set, it overrides auto-derive
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "AGRT-001",
			Statement:         "Governance rule",
			ClaimType:         "meta",     // auto-derive would give "empirical"
			Domain:            "agent_rights",
			EpistemicCategory: "protocol", // explicit override
			Confidence:        1.0,
		},
	}

	facts := AxiomsToFacts(axioms)
	if facts[0].EpistemicCategory != "protocol" {
		t.Errorf("explicit override: category = %q, want protocol", facts[0].EpistemicCategory)
	}
}

func TestResolveAxiomCategory(t *testing.T) {
	tests := []struct {
		name             string
		claimType        string
		explicitCategory string
		wantCategory     string
		wantDisagrees    bool
	}{
		{"axiom auto", "axiom", "", "analytic", false},
		{"definition auto", "definition", "", "analytic", false},
		{"empirical_axiom auto", "empirical_axiom", "", "empirical", false},
		{"derived_claim auto", "derived_claim", "", "formal", false},
		{"measurement_fact auto", "measurement_fact", "", "empirical", false},
		{"regime_declaration auto", "regime_declaration", "", "formal", false},
		{"meta auto", "meta", "", "empirical", false},
		{"explicit agrees", "axiom", "analytic", "analytic", false},
		{"explicit disagrees", "meta", "protocol", "protocol", true},
		{"explicit no claim type", "", "formal", "formal", false},
		{"legacy no type no category", "", "", "analytic", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &GenesisAxiom{
				ClaimType:         tt.claimType,
				EpistemicCategory: tt.explicitCategory,
			}
			got, disagrees := ResolveAxiomCategory(a)
			if got != tt.wantCategory {
				t.Errorf("category = %q, want %q", got, tt.wantCategory)
			}
			if disagrees != tt.wantDisagrees {
				t.Errorf("disagrees = %v, want %v", disagrees, tt.wantDisagrees)
			}
		})
	}
}

func TestDerivedConfidenceValidation(t *testing.T) {
	t.Run("valid: derived <= min dep", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "empirical_axiom",
				Domain: "mathematics", EpistemicCategory: "empirical", Confidence: 0.95},
			{AxiomID: "MATH-003", Statement: "C", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.90,
				Dependencies: []string{"MATH-001", "MATH-002"}},
		}
		err := ValidateDerivedConfidence(axioms)
		if err != nil {
			t.Errorf("should be valid (0.90 <= 0.95): %v", err)
		}
	})

	t.Run("valid: derived == min dep", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 0.95},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.95,
				Dependencies: []string{"MATH-001"}},
		}
		err := ValidateDerivedConfidence(axioms)
		if err != nil {
			t.Errorf("should be valid (0.95 == 0.95): %v", err)
		}
	})

	t.Run("invalid: derived > min dep", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 0.90},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.95,
				Dependencies: []string{"MATH-001"}},
		}
		err := ValidateDerivedConfidence(axioms)
		if err == nil {
			t.Error("expected error: derived 0.95 > dep 0.90")
		}
	})

	t.Run("non-derived claims skip check", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 0.50},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0,
				Dependencies: []string{"MATH-001"}},
		}
		err := ValidateDerivedConfidence(axioms)
		if err != nil {
			t.Errorf("non-derived claims should skip confidence check: %v", err)
		}
	})

	t.Run("derived with default confidence", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal",
				Dependencies: []string{"MATH-001"}},
		}
		err := ValidateDerivedConfidence(axioms)
		if err != nil {
			t.Errorf("default 0.95 <= 1.0 should be valid: %v", err)
		}
	})
}

func TestValidateAxioms_DerivedConfidenceTooHigh(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
			Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 0.80},
		{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim",
			Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.90,
			Dependencies: []string{"MATH-001"}},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	if err == nil {
		t.Error("expected derived confidence validation error from ValidateAxioms")
	}
}

func TestLoadTemplateAxioms(t *testing.T) {
	// Load the actual template file and validate it
	templatePath := "../../../scripts/genesis-axioms-template.json"
	axioms, err := LoadAxiomsFromFile(templatePath)
	if err != nil {
		t.Skipf("Skipping template validation (file not at %s): %v", templatePath, err)
	}

	domains := DefaultDomains()
	domainNames := make([]string, len(domains))
	for i, d := range domains {
		domainNames[i] = d.Name
	}

	if err := ValidateAxioms(axioms, domainNames); err != nil {
		t.Errorf("Template axioms failed validation: %v", err)
	}

	if len(axioms) != 75 {
		t.Errorf("Template has %d axioms, want 75", len(axioms))
	}
}

func TestAxiomsToFactReferences(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-003", Statement: "C", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-001", "MATH-002"}},
		{AxiomID: "PHYS-001", Statement: "D", Domain: "physics", EpistemicCategory: "empirical",
			Dependencies: []string{"MATH-001"}},
	}

	refs := AxiomsToFactReferences(axioms)

	// MATH-001 has 0 deps, MATH-002 has 1, MATH-003 has 2, PHYS-001 has 1 → total 4
	if len(refs) != 4 {
		t.Fatalf("got %d references, want 4", len(refs))
	}

	// All should be "supports" type at full weight
	for i, r := range refs {
		if r.ReferenceType != "supports" {
			t.Errorf("ref[%d] type = %q, want supports", i, r.ReferenceType)
		}
		if r.Weight != 1_000_000 {
			t.Errorf("ref[%d] weight = %d, want 1000000", i, r.Weight)
		}
		if r.CreatedAtHeight != 0 {
			t.Errorf("ref[%d] created_at_height = %d, want 0", i, r.CreatedAtHeight)
		}
	}
}

func TestFactReferenceDirection(t *testing.T) {
	// "A depends on B" → from=A, to=B
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "Base", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "Derived", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-001"}},
	}

	refs := AxiomsToFactReferences(axioms)
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}

	// MATH-002 depends on MATH-001 → from=MATH-002 (dependent), to=MATH-001 (dependency)
	if refs[0].FromFactId != "MATH-002" {
		t.Errorf("from_fact_id = %q, want MATH-002", refs[0].FromFactId)
	}
	if refs[0].ToFactId != "MATH-001" {
		t.Errorf("to_fact_id = %q, want MATH-001", refs[0].ToFactId)
	}
}

func TestFactReferenceTypes_AllSupports(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-001"}},
		{AxiomID: "LOGIC-001", Statement: "C", Domain: "logic", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
	}

	refs := AxiomsToFactReferences(axioms)
	for i, r := range refs {
		if r.ReferenceType != "supports" {
			t.Errorf("ref[%d] type = %q, want supports", i, r.ReferenceType)
		}
	}
}

func TestAxiomsToFactReferences_NoDeps(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "LOGIC-001", Statement: "B", Domain: "logic", EpistemicCategory: "analytic"},
	}

	refs := AxiomsToFactReferences(axioms)
	if len(refs) != 0 {
		t.Errorf("got %d refs for axioms with no dependencies, want 0", len(refs))
	}
}

func TestComputeFactReferenceStats(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-001"}},
		{AxiomID: "PHYS-001", Statement: "C", Domain: "physics", EpistemicCategory: "empirical",
			Dependencies: []string{"MATH-001"}},
	}

	refs := AxiomsToFactReferences(axioms)
	stats := ComputeFactReferenceStats(axioms, refs)

	if stats.Total != 2 {
		t.Errorf("Total = %d, want 2", stats.Total)
	}
	if stats.ByType["supports"] != 2 {
		t.Errorf("supports count = %d, want 2", stats.ByType["supports"])
	}
	// MATH-002→MATH-001 is intra-domain (both mathematics)
	// PHYS-001→MATH-001 is cross-domain (physics→mathematics)
	if stats.IntraDomain != 1 {
		t.Errorf("IntraDomain = %d, want 1", stats.IntraDomain)
	}
	if stats.CrossDomain != 1 {
		t.Errorf("CrossDomain = %d, want 1", stats.CrossDomain)
	}
}

func TestGenesisValidate_FactReferences(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
			{Id: "MATH-002", Domain: "mathematics", EpistemicCategory: "formal", References: []string{"MATH-001"}},
		},
		FactReferences: []*FactReference{
			{FromFactId: "MATH-002", ToFactId: "MATH-001", ReferenceType: "supports", Weight: 1_000_000},
		},
	}
	if err := gs.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenesisValidate_FactReferences_BadFrom(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
		},
		FactReferences: []*FactReference{
			{FromFactId: "NONEXISTENT", ToFactId: "MATH-001", ReferenceType: "supports"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected error for bad from_fact_id")
	}
}

func TestGenesisValidate_FactReferences_BadTo(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
		},
		FactReferences: []*FactReference{
			{FromFactId: "MATH-001", ToFactId: "NONEXISTENT", ReferenceType: "supports"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected error for bad to_fact_id")
	}
}

func TestGenesisValidate_FactReferences_EmptyFields(t *testing.T) {
	gs := &GenesisState{
		Domains: DefaultDomains(),
		Facts: []*Fact{
			{Id: "MATH-001", Domain: "mathematics", EpistemicCategory: "analytic"},
		},
		FactReferences: []*FactReference{
			{FromFactId: "", ToFactId: "MATH-001", ReferenceType: "supports"},
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("expected error for empty from_fact_id")
	}
}

func TestComputeDAGStats(t *testing.T) {
	// DAG structure:
	//   A (root, orphan)
	//   B (root) → D depends on B
	//   C (root) → D depends on C
	//   D (depth 1) → E depends on D
	//   E (depth 2, leaf)
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-003", Statement: "C", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-004", Statement: "D", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-002", "MATH-003"}},
		{AxiomID: "MATH-005", Statement: "E", Domain: "mathematics", EpistemicCategory: "formal",
			Dependencies: []string{"MATH-004"}},
	}

	stats, err := ComputeDAGStats(axioms)
	if err != nil {
		t.Fatalf("ComputeDAGStats: %v", err)
	}

	// 3 roots: A, B, C
	if stats.RootCount != 3 {
		t.Errorf("RootCount = %d, want 3", stats.RootCount)
	}

	// Max depth = 2 (B/C→D→E)
	if stats.MaxDepth != 2 {
		t.Errorf("MaxDepth = %d, want 2", stats.MaxDepth)
	}

	// Depth histogram: 0→3 (A,B,C), 1→1 (D), 2→1 (E)
	if stats.DepthCounts[0] != 3 {
		t.Errorf("DepthCounts[0] = %d, want 3", stats.DepthCounts[0])
	}
	if stats.DepthCounts[1] != 1 {
		t.Errorf("DepthCounts[1] = %d, want 1", stats.DepthCounts[1])
	}
	if stats.DepthCounts[2] != 1 {
		t.Errorf("DepthCounts[2] = %d, want 1", stats.DepthCounts[2])
	}

	// Total edges = 3 (D→B, D→C, E→D)
	if stats.TotalEdges != 3 {
		t.Errorf("TotalEdges = %d, want 3", stats.TotalEdges)
	}

	// Orphan: A (no deps, nothing depends on it)
	if len(stats.Orphans) != 1 || stats.Orphans[0] != "MATH-001" {
		t.Errorf("Orphans = %v, want [MATH-001]", stats.Orphans)
	}

	// MostReferenced should include MATH-002, MATH-003 (each 1 dependent), MATH-004 (1 dependent)
	if len(stats.MostReferenced) != 3 {
		t.Errorf("MostReferenced has %d entries, want 3", len(stats.MostReferenced))
	}
}

func TestComputeDAGStats_Cycle(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-002"}},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
	}
	_, err := ComputeDAGStats(axioms)
	if err == nil {
		t.Error("expected cycle error")
	}
}

func TestCrossDomainMatrix(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "MATH-002", Statement: "B", Domain: "mathematics", EpistemicCategory: "analytic"},
		{AxiomID: "PHYS-001", Statement: "C", Domain: "physics", EpistemicCategory: "empirical",
			Dependencies: []string{"MATH-001", "MATH-002"}},
		{AxiomID: "LOGIC-001", Statement: "D", Domain: "logic", EpistemicCategory: "analytic",
			Dependencies: []string{"MATH-001"}},
	}

	matrix := ComputeCrossDomainMatrix(axioms)

	// Should have 2 domains in entries: physics→mathematics (2), logic→mathematics (1)
	if len(matrix.Entries) != 2 {
		t.Fatalf("matrix entries = %d, want 2", len(matrix.Entries))
	}

	// First entry (highest count): physics→mathematics = 2
	if matrix.Entries[0].FromDomain != "physics" || matrix.Entries[0].ToDomain != "mathematics" {
		t.Errorf("entry[0] = %s→%s, want physics→mathematics",
			matrix.Entries[0].FromDomain, matrix.Entries[0].ToDomain)
	}
	if matrix.Entries[0].Count != 2 {
		t.Errorf("entry[0].Count = %d, want 2", matrix.Entries[0].Count)
	}

	// Second entry: logic→mathematics = 1
	if matrix.Entries[1].FromDomain != "logic" || matrix.Entries[1].ToDomain != "mathematics" {
		t.Errorf("entry[1] = %s→%s, want logic→mathematics",
			matrix.Entries[1].FromDomain, matrix.Entries[1].ToDomain)
	}
	if matrix.Entries[1].Count != 1 {
		t.Errorf("entry[1].Count = %d, want 1", matrix.Entries[1].Count)
	}

	// 3 unique domains
	if len(matrix.Domains) != 3 {
		t.Errorf("domains = %d, want 3", len(matrix.Domains))
	}
}

func TestValidateStratumConsistency(t *testing.T) {
	t.Run("valid: axiom depends on axiom", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic"},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic",
				Dependencies: []string{"MATH-001"}},
		}
		if err := ValidateStratumConsistency(axioms); err != nil {
			t.Errorf("should be valid: %v", err)
		}
	})

	t.Run("valid: derived_claim depends on axiom", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic"},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal",
				Dependencies: []string{"MATH-001"}},
		}
		if err := ValidateStratumConsistency(axioms); err != nil {
			t.Errorf("should be valid: %v", err)
		}
	})

	t.Run("invalid: axiom depends on derived_claim", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "MATH-001", Statement: "A", ClaimType: "derived_claim",
				Domain: "mathematics", EpistemicCategory: "formal"},
			{AxiomID: "MATH-002", Statement: "B", ClaimType: "axiom",
				Domain: "mathematics", EpistemicCategory: "analytic",
				Dependencies: []string{"MATH-001"}},
		}
		err := ValidateStratumConsistency(axioms)
		if err == nil {
			t.Error("expected stratum violation error")
		}
		if err != nil && !strings.Contains(err.Error(), "stratum violation") {
			t.Errorf("expected 'stratum violation' in error, got: %v", err)
		}
	})

	t.Run("valid: definition depends on measurement_fact (allowed)", func(t *testing.T) {
		// Definitions MAY depend on derived types — only axioms are strictly foundational
		axioms := []*GenesisAxiom{
			{AxiomID: "PHYS-001", Statement: "A", ClaimType: "measurement_fact",
				Domain: "physics", EpistemicCategory: "empirical"},
			{AxiomID: "PHYS-002", Statement: "B", ClaimType: "definition",
				Domain: "physics", EpistemicCategory: "analytic",
				Dependencies: []string{"PHYS-001"}},
		}
		if err := ValidateStratumConsistency(axioms); err != nil {
			t.Errorf("definition depending on measurement_fact should be allowed: %v", err)
		}
	})

	t.Run("invalid: axiom depends on measurement_fact", func(t *testing.T) {
		axioms := []*GenesisAxiom{
			{AxiomID: "PHYS-001", Statement: "A", ClaimType: "measurement_fact",
				Domain: "physics", EpistemicCategory: "empirical"},
			{AxiomID: "PHYS-002", Statement: "B", ClaimType: "axiom",
				Domain: "physics", EpistemicCategory: "analytic",
				Dependencies: []string{"PHYS-001"}},
		}
		err := ValidateStratumConsistency(axioms)
		if err == nil {
			t.Error("expected stratum violation: axiom depends on measurement_fact")
		}
	})
}

func TestDAGStatsWithAllDomains(t *testing.T) {
	// Load the actual axioms-777.json file; skip if not present
	axiomPath := "../../../scripts/axioms-777.json"
	axioms, err := LoadAxiomsFromFile(axiomPath)
	if err != nil {
		t.Skipf("Skipping (file not at %s): %v", axiomPath, err)
	}

	stats, err := ComputeDAGStats(axioms)
	if err != nil {
		t.Fatalf("ComputeDAGStats on axioms-777: %v", err)
	}

	if stats.RootCount == 0 {
		t.Error("RootCount should be > 0")
	}
	if stats.MaxDepth == 0 {
		t.Error("MaxDepth should be > 0")
	}
	if stats.TotalEdges == 0 {
		t.Error("TotalEdges should be > 0")
	}
	if len(stats.MostReferenced) == 0 {
		t.Error("MostReferenced should be non-empty")
	}

	t.Logf("DAG Stats: roots=%d, maxDepth=%d, edges=%d, orphans=%d",
		stats.RootCount, stats.MaxDepth, stats.TotalEdges, len(stats.Orphans))
	for d := 0; d <= stats.MaxDepth; d++ {
		t.Logf("  depth %d: %d items", d, stats.DepthCounts[d])
	}

	matrix := ComputeCrossDomainMatrix(axioms)
	if len(matrix.Entries) == 0 {
		t.Error("CrossDomainMatrix should have entries")
	}
	t.Logf("Cross-domain entries: %d, domains: %d", len(matrix.Entries), len(matrix.Domains))
}

func TestValidateAxiomDAG_EnhancedMissingDepError(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "PHYS-001", Statement: "A", Domain: "physics", EpistemicCategory: "empirical",
			Dependencies: []string{"MATH-999"}},
	}
	err := ValidateAxiomDAG(axioms)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	// Should include domain hint from prefix
	if !strings.Contains(err.Error(), "mathematics") {
		t.Errorf("error should mention expected domain 'mathematics', got: %v", err)
	}
	if !strings.Contains(err.Error(), "MATH") {
		t.Errorf("error should mention prefix 'MATH', got: %v", err)
	}
}

// ---------- FARM-12: Stratum Types Tests ----------

func TestValidStrata(t *testing.T) {
	expectedStrata := []string{"fundamental", "physical", "chemical", "biological", "cognitive", "social", "technological"}
	if len(ValidStrata) != 7 {
		t.Errorf("ValidStrata has %d entries, want 7", len(ValidStrata))
	}
	for _, s := range expectedStrata {
		if !ValidStrata[s] {
			t.Errorf("ValidStrata missing %q", s)
		}
	}
}

func TestBootstrapDomainStrataComplete(t *testing.T) {
	domains := DefaultDomains()
	if len(BootstrapDomainStrata) != len(domains) {
		t.Errorf("BootstrapDomainStrata has %d entries, want %d (matching DefaultDomains)",
			len(BootstrapDomainStrata), len(domains))
	}

	// Every genesis domain must have a stratum assignment
	for _, d := range domains {
		stratum, ok := BootstrapDomainStrata[d.Name]
		if !ok {
			t.Errorf("domain %q missing from BootstrapDomainStrata", d.Name)
			continue
		}
		// Assigned stratum must be valid
		if !ValidStrata[stratum] {
			t.Errorf("domain %q has invalid stratum %q", d.Name, stratum)
		}
	}
}

func TestStratumStakeMultiplierComplete(t *testing.T) {
	// Every valid stratum must have a stake multiplier
	for stratum := range ValidStrata {
		if _, ok := StratumStakeMultiplier[stratum]; !ok {
			t.Errorf("stratum %q missing from StratumStakeMultiplier", stratum)
		}
	}
	// Multipliers should be monotonically non-decreasing as strata get more fundamental
	if StratumStakeMultiplier["fundamental"] <= StratumStakeMultiplier["social"] {
		t.Error("fundamental multiplier should be > social multiplier")
	}
}

func TestDefaultProposalStratum(t *testing.T) {
	if DefaultProposalStratum != "social" {
		t.Errorf("DefaultProposalStratum = %q, want social", DefaultProposalStratum)
	}
	if !ValidStrata[DefaultProposalStratum] {
		t.Errorf("DefaultProposalStratum %q is not a valid stratum", DefaultProposalStratum)
	}
}
