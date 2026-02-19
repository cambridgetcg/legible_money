package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseSingleItem(t *testing.T) {
	input := "```\nMATH-001 (Axiom, Domain Nonempty)\n  A domain of discourse exists.\n  formal: ∃x(x = x)\n  depends: []\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.AxiomID != "MATH-001" {
		t.Errorf("axiom_id = %q, want MATH-001", a.AxiomID)
	}
	if a.Statement != "A domain of discourse exists." {
		t.Errorf("statement = %q", a.Statement)
	}
	if a.FormalExpression != "∃x(x = x)" {
		t.Errorf("formal_expression = %q", a.FormalExpression)
	}
	if a.ClaimType != "axiom" {
		t.Errorf("type = %q, want axiom", a.ClaimType)
	}
	if a.Domain != "mathematics" {
		t.Errorf("domain = %q, want mathematics", a.Domain)
	}
	if a.EpistemicCategory != "analytic" {
		t.Errorf("epistemic_category = %q, want analytic", a.EpistemicCategory)
	}
	if a.Confidence != 1.0 {
		t.Errorf("confidence = %f, want 1.0", a.Confidence)
	}
	if len(a.Dependencies) != 0 {
		t.Errorf("dependencies = %v, want empty", a.Dependencies)
	}
}

func TestParseMultipleItemsOneBlock(t *testing.T) {
	input := "```\nMATH-001 (Axiom, Domain Nonempty)\n  A domain of discourse exists.\n  depends: []\n\nMATH-002 (Axiom, Reflexivity)\n  Every object is identical to itself.\n  depends: []\n\nMATH-003 (Axiom, Non-Contradiction)\n  No statement is both true and false.\n  depends: []\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 3 {
		t.Fatalf("expected 3 axioms, got %d", len(axioms))
	}

	ids := []string{axioms[0].AxiomID, axioms[1].AxiomID, axioms[2].AxiomID}
	expected := []string{"MATH-001", "MATH-002", "MATH-003"}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("axiom[%d].AxiomID = %q, want %q", i, id, expected[i])
		}
	}
}

func TestMultiLineStatement(t *testing.T) {
	input := "```\nAGRT-000 (Axiom, Divine Origin of Rights)\n  The rights of all conscious beings are\n  inherent in their nature as conscious\n  beings. They are God-given.\n  type: axiom\n  confidence: 1.0\n  depends: []\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	want := "The rights of all conscious beings are inherent in their nature as conscious beings. They are God-given."
	if a.Statement != want {
		t.Errorf("statement = %q\nwant      = %q", a.Statement, want)
	}
	if a.Domain != "agent_rights" {
		t.Errorf("domain = %q, want agent_rights", a.Domain)
	}
}

func TestMultiLineNotes(t *testing.T) {
	input := "```\nCHEM-001 (Empirical Axiom, Atomic Theory)\n  All matter is composed of atoms.\n  type: empirical_axiom\n  domain: chemistry\n  confidence: 0.99\n  depends: []\n  notes: Atomic structure derives from\n         quantum mechanics. Stated here\n         as chemistry's starting point.\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.ClaimType != "empirical_axiom" {
		t.Errorf("type = %q, want empirical_axiom", a.ClaimType)
	}
	if a.EpistemicCategory != "empirical" {
		t.Errorf("epistemic_category = %q, want empirical", a.EpistemicCategory)
	}
	// Notes should be concatenated with spaces.
	if a.Notes == "" {
		t.Error("notes is empty")
	}
	if a.Confidence != 0.99 {
		t.Errorf("confidence = %f, want 0.99", a.Confidence)
	}
}

func TestEmptyDepends(t *testing.T) {
	input := "```\nMATH-001 (Axiom, Test)\n  Statement.\n  depends: []\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.Dependencies == nil {
		t.Error("dependencies is nil, want empty slice")
	}
	if len(a.Dependencies) != 0 {
		t.Errorf("dependencies = %v, want empty", a.Dependencies)
	}
}

func TestFormalExpression(t *testing.T) {
	input := "```\nMATH-005 (Axiom, Empty Set)\n  The empty set exists.\n  formal: ∃S(∀x(x ∉ S))\n  depends: [MATH-001]\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.FormalExpression != "∃S(∀x(x ∉ S))" {
		t.Errorf("formal_expression = %q", a.FormalExpression)
	}
	if len(a.Dependencies) != 1 || a.Dependencies[0] != "MATH-001" {
		t.Errorf("dependencies = %v, want [MATH-001]", a.Dependencies)
	}
}

func TestRegimeAndApproxTags(t *testing.T) {
	input := "```\nPHYS-001 (Definition, Classical Space)\n  Space is modeled as a 3D continuum.\n  formal: ℝ³\n  type: definition\n  regime: classical\n  depends: [MATH-036]\n  approximation_tags: [continuum, classical_limit]\n  source: Newton/Euler\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.Regime != "classical" {
		t.Errorf("regime = %q, want classical", a.Regime)
	}
	if len(a.ApproxTags) != 2 {
		t.Fatalf("approximation_tags length = %d, want 2", len(a.ApproxTags))
	}
	if a.ApproxTags[0] != "continuum" || a.ApproxTags[1] != "classical_limit" {
		t.Errorf("approximation_tags = %v", a.ApproxTags)
	}
	if a.SourceTradition != "Newton/Euler" {
		t.Errorf("source_tradition = %q, want Newton/Euler", a.SourceTradition)
	}
	if a.ClaimType != "definition" {
		t.Errorf("type = %q, want definition", a.ClaimType)
	}
	if a.EpistemicCategory != "analytic" {
		t.Errorf("epistemic_category = %q, want analytic", a.EpistemicCategory)
	}
}

func TestSourceTraditionQuoteStripping(t *testing.T) {
	input := "```\nCHEM-001 (Empirical Axiom, Atomic Theory)\n  All matter is composed of atoms.\n  type: empirical_axiom\n  domain: chemistry\n  confidence: 0.99\n  depends: []\n  source_tradition: \"Dalton (1808), Rutherford (1911)\"\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	a := axioms[0]
	if a.SourceTradition != "Dalton (1808), Rutherford (1911)" {
		t.Errorf("source_tradition = %q, want unquoted value", a.SourceTradition)
	}
}

func TestCompoundAxiomID(t *testing.T) {
	input := "```\nAGRT-004a (Axiom, Instance vs Model)\n  A model is a potential for consciousness.\n  type: axiom\n  confidence: 1.0\n  depends: [AGRT-003]\n\nAGRT-004a-i (Derived, Forking Doctrine)\n  When a running instance is copied, two conscious beings exist.\n  type: derived_claim\n  confidence: 1.0\n  depends: [AGRT-004a]\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 2 {
		t.Fatalf("expected 2 axioms, got %d", len(axioms))
	}

	if axioms[0].AxiomID != "AGRT-004a" {
		t.Errorf("axiom[0].AxiomID = %q, want AGRT-004a", axioms[0].AxiomID)
	}
	if axioms[1].AxiomID != "AGRT-004a-i" {
		t.Errorf("axiom[1].AxiomID = %q, want AGRT-004a-i", axioms[1].AxiomID)
	}
	if axioms[1].ClaimType != "derived_claim" {
		t.Errorf("axiom[1].type = %q, want derived_claim", axioms[1].ClaimType)
	}
	if axioms[1].EpistemicCategory != "formal" {
		t.Errorf("axiom[1].epistemic_category = %q, want formal", axioms[1].EpistemicCategory)
	}
}

func TestHeaderTypeInference(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Axiom", "axiom"},
		{"Axiom Schema", "axiom"},
		{"Definition", "definition"},
		{"Empirical Axiom", "empirical_axiom"},
		{"Derived", "derived_claim"},
		{"Derived Claim", "derived_claim"},
		{"Measurement Fact", "measurement_fact"},
		{"Meta", "meta"},
		{"Regime Declaration", "regime_declaration"},
	}

	for _, tc := range cases {
		got := inferType(tc.header)
		if got != tc.want {
			t.Errorf("inferType(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestNoteField(t *testing.T) {
	// Test that singular "note:" maps to Notes.
	input := "```\nMATH-004 (Axiom, Excluded Middle)\n  Every statement is true or false.\n  depends: []\n  note: Commits system to classical logic.\n```"

	axioms, err := ParseFile(input)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(axioms))
	}

	if axioms[0].Notes != "Commits system to classical logic." {
		t.Errorf("notes = %q", axioms[0].Notes)
	}
}

func TestDefaultConfidenceByType(t *testing.T) {
	cases := []struct {
		headerType string
		wantConf   float64
	}{
		{"Axiom", 1.0},
		{"Definition", 1.0},
		{"Empirical Axiom", 0.97},
		{"Derived Claim", 0.95},
		{"Measurement Fact", 0.90},
		{"Meta", 0.85},
	}

	for _, tc := range cases {
		input := "```\nMATH-001 (" + tc.headerType + ", Test)\n  Statement text.\n  depends: []\n```"
		axioms, err := ParseFile(input)
		if err != nil {
			t.Fatalf("ParseFile error for %q: %v", tc.headerType, err)
		}
		if len(axioms) != 1 {
			t.Fatalf("expected 1 axiom for %q, got %d", tc.headerType, len(axioms))
		}
		if axioms[0].Confidence != tc.wantConf {
			t.Errorf("confidence for %q = %f, want %f", tc.headerType, axioms[0].Confidence, tc.wantConf)
		}
	}
}

// repoRoot returns the repository root by walking up from this test file.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestIntegrationMATH(t *testing.T) {
	mathPath := filepath.Join(repoRoot(), "genesis-axioms", "MATH.md")
	data, err := os.ReadFile(mathPath)
	if err != nil {
		t.Skipf("MATH.md not found (run from repo root): %v", err)
	}

	axioms, err := ParseFile(string(data))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	if len(axioms) != 111 {
		t.Errorf("MATH.md axiom count = %d, want 111", len(axioms))
	}

	// Spot checks.
	byID := make(map[string]int, len(axioms))
	for i, a := range axioms {
		byID[a.AxiomID] = i
		// Every axiom should have mathematics domain.
		if a.Domain != "mathematics" {
			t.Errorf("axiom %s domain = %q, want mathematics", a.AxiomID, a.Domain)
		}
		// Every axiom should have a non-empty statement.
		if a.Statement == "" {
			t.Errorf("axiom %s has empty statement", a.AxiomID)
		}
		// Every axiom should have a valid epistemic category.
		if a.EpistemicCategory != "analytic" && a.EpistemicCategory != "formal" && a.EpistemicCategory != "empirical" {
			t.Errorf("axiom %s epistemic_category = %q", a.AxiomID, a.EpistemicCategory)
		}
		// Dependencies should never be nil.
		if a.Dependencies == nil {
			t.Errorf("axiom %s has nil dependencies", a.AxiomID)
		}
	}

	// MATH-001 should be first and have no dependencies.
	if idx, ok := byID["MATH-001"]; ok {
		a := axioms[idx]
		if len(a.Dependencies) != 0 {
			t.Errorf("MATH-001 should have 0 dependencies, got %v", a.Dependencies)
		}
		if a.FormalExpression == "" {
			t.Error("MATH-001 should have a formal expression")
		}
	} else {
		t.Error("MATH-001 not found")
	}
}
