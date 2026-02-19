package types

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		{"ETHICS-001", false},
		{"INFO-123", false},
		// Invalid
		{"MATH001", true},    // no dash
		{"math-001", true},   // lowercase
		{"MATH-01", true},    // only 2 digits
		{"MATH-0001", true},  // 4 digits
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
	if len(DomainPrefixMap) != 15 {
		t.Errorf("DomainPrefixMap has %d entries, want 15", len(DomainPrefixMap))
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
	if len(domains) != 16 {
		t.Errorf("DefaultDomains() has %d domains, want 16", len(domains))
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
	if len(names) != 15 {
		t.Errorf("AxiomDomainNames() has %d entries, want 15", len(names))
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
