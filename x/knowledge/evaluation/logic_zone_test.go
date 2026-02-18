package evaluation

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func TestEvaluateLogicZone_Propositional(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, zone string) uint64 {
			if zone == "propositional" {
				return 1_000_000 // 100%
			}
			return 700_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return false
		},
	}
	claim := &types.Claim{LogicZone: "propositional"}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.Ceiling != 1_000_000 {
		t.Errorf("expected 1000000, got %d", result.Ceiling)
	}
	if result.ForceVerdict != "" {
		t.Errorf("expected no forced verdict for propositional, got %s", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_Peano_NoGrounding(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 850_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano"
		},
		GetFact: func(_ sdk.Context, _ string) (*types.Fact, bool) {
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "peano", References: nil}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.Ceiling != 850_000 {
		t.Errorf("expected 850000, got %d", result.Ceiling)
	}
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected forced abstain for Godel zone without grounding, got %q", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_Peano_WithGrounding(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 850_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano"
		},
		GetFact: func(_ sdk.Context, id string) (*types.Fact, bool) {
			if id == "axiom-1" {
				// Propositional fact (more formal) grounds a Peano claim — valid
				return &types.Fact{
					Id: "axiom-1", Status: types.FactStatusVerified,
					Confidence: 950_000, LogicZone: "propositional", Domain: "mathematics",
				}, true
			}
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "peano", Domain: "mathematics", References: []string{"axiom-1"}}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.ForceVerdict != "" {
		t.Errorf("expected no forced verdict when grounded with compatible zone, got %q", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_Peano_EmpiricalRefRejected(t *testing.T) {
	// G-2: An empirical fact should NOT ground a Peano claim
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 850_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano"
		},
		GetFact: func(_ sdk.Context, id string) (*types.Fact, bool) {
			if id == "obs-1" {
				return &types.Fact{
					Id: "obs-1", Status: types.FactStatusVerified,
					Confidence: 700_000, LogicZone: "empirical", Domain: "mathematics",
				}, true
			}
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "peano", Domain: "mathematics", References: []string{"obs-1"}}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("empirical fact should not ground Peano claim, expected abstain, got %q", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_Peano_WrongDomainRejected(t *testing.T) {
	// G-2: A fact in a different domain should NOT ground a Gödelian claim
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 850_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano"
		},
		GetFact: func(_ sdk.Context, id string) (*types.Fact, bool) {
			if id == "axiom-1" {
				return &types.Fact{
					Id: "axiom-1", Status: types.FactStatusVerified,
					Confidence: 950_000, LogicZone: "propositional", Domain: "physics",
				}, true
			}
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "peano", Domain: "mathematics", References: []string{"axiom-1"}}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("cross-domain ref should not ground Peano claim, expected abstain, got %q", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_Peano_EmptyZoneRefAllowed(t *testing.T) {
	// Backward compat: fact with no LogicZone set still grounds (no restriction)
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 850_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano"
		},
		GetFact: func(_ sdk.Context, id string) (*types.Fact, bool) {
			if id == "legacy-fact" {
				return &types.Fact{
					Id: "legacy-fact", Status: types.FactStatusVerified,
					Confidence: 900_000, LogicZone: "", Domain: "",
				}, true
			}
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "peano", Domain: "mathematics", References: []string{"legacy-fact"}}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.ForceVerdict != "" {
		t.Errorf("legacy fact (no zone) should still ground, got %q", result.ForceVerdict)
	}
}

func TestIsZoneCompatibleGrounding(t *testing.T) {
	tests := []struct {
		refZone   string
		claimZone string
		want      bool
	}{
		{"propositional", "peano", true},         // more formal → less formal: OK
		{"propositional", "propositional", true}, // same zone: OK
		{"presburger", "peano", true},            // more formal → less formal: OK
		{"peano", "peano", true},                 // same zone: OK
		{"empirical", "peano", false},            // less formal → more formal: BLOCKED
		{"empirical", "set_theory", false},       // less formal → more formal: BLOCKED
		{"set_theory", "peano", false},           // less formal → more formal: BLOCKED
		{"peano", "set_theory", true},            // more formal → less formal: OK
		{"", "peano", true},                      // empty ref zone: backward compat
		{"peano", "", true},                      // empty claim zone: backward compat
		{"unknown", "peano", true},               // unknown zone: forward compat
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s", tt.refZone, tt.claimZone)
		t.Run(name, func(t *testing.T) {
			got := isZoneCompatibleGrounding(tt.refZone, tt.claimZone)
			if got != tt.want {
				t.Errorf("isZoneCompatibleGrounding(%q, %q) = %v, want %v",
					tt.refZone, tt.claimZone, got, tt.want)
			}
		})
	}
}

func TestEvaluateLogicZone_SetTheory(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, _ string) uint64 {
			return 800_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "set_theory"
		},
		GetFact: func(_ sdk.Context, _ string) (*types.Fact, bool) {
			return nil, false
		},
	}
	claim := &types.Claim{LogicZone: "set_theory"}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.Ceiling != 800_000 {
		t.Errorf("expected 800000, got %d", result.Ceiling)
	}
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected forced abstain, got %q", result.ForceVerdict)
	}
}

func TestEvaluateLogicZone_EmptyZone(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{ExtendedParams: types.DefaultExtendedParams()}
	claim := &types.Claim{LogicZone: ""}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	// Empty zone: default 1,000,000
	if result.Ceiling != 1_000_000 {
		t.Errorf("expected 1000000 for empty zone, got %d", result.Ceiling)
	}
}

func TestEvaluateLogicZone_Empirical(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetZoneConfidenceCeiling: func(_ sdk.Context, zone string) uint64 {
			if zone == "empirical" {
				return 700_000
			}
			return 1_000_000
		},
		RequiresIncompletenessAck: func(_ sdk.Context, _ string) bool {
			return false
		},
	}
	claim := &types.Claim{LogicZone: "empirical"}

	result := evaluateLogicZone(ctx, evalCtx, claim)
	if result.Ceiling != 700_000 {
		t.Errorf("expected 700000, got %d", result.Ceiling)
	}
	if result.ForceVerdict != "" {
		t.Errorf("empirical zone should not force verdict, got %q", result.ForceVerdict)
	}
}

func TestApplyCeilings_TripleCeiling(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetConfidenceCeiling: func(_ sdk.Context, domain string) (uint64, uint64, error) {
			if domain == "sociology" {
				return 850_000, 0, nil // social stratum ceiling
			}
			return 0, 0, fmt.Errorf("not found")
		},
	}

	claim := &types.Claim{
		Domain:            "sociology",
		EpistemicCategory: "social",
		LogicZone:         "empirical",
	}

	// Zone ceiling from stage 4 = 700,000 (empirical)
	// Stratum ceiling = 850,000 (social stratum from ontology)
	// Category ceiling = 850,000 (CATEGORY_DECAY["social"].MaxConfidence)
	result := applyCeilings(ctx, evalCtx, claim, 700_000)

	// min(700,000, 850,000, 850,000) = 700,000
	if result.Ceiling != 700_000 {
		t.Errorf("expected 700000 (zone wins), got %d", result.Ceiling)
	}
}

func TestApplyCeilings_CategoryCeilingWins(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetConfidenceCeiling: func(_ sdk.Context, _ string) (uint64, uint64, error) {
			return 1_000_000, 0, nil // fundamental stratum
		},
	}

	claim := &types.Claim{
		Domain:            "mathematics",
		EpistemicCategory: "predictive", // MaxConfidence = 900,000
		LogicZone:         "propositional",
	}

	// Zone ceiling = 1,000,000 (propositional)
	// Stratum ceiling = 1,000,000 (fundamental)
	// Category ceiling = 900,000 (predictive)
	result := applyCeilings(ctx, evalCtx, claim, 1_000_000)

	if result.Ceiling != 900_000 {
		t.Errorf("expected 900000 (category wins), got %d", result.Ceiling)
	}
}

func TestApplyCeilings_FallbackStratum(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams:       types.DefaultExtendedParams(),
		GetConfidenceCeiling: nil, // no ontology keeper
	}

	claim := &types.Claim{
		Domain:            "biology",
		EpistemicCategory: "empirical",
	}

	// Zone = 1,000,000 (no zone specified)
	// Stratum fallback: biology → biological → 950,000
	// Category: empirical → MaxConfidence = 990,000
	result := applyCeilings(ctx, evalCtx, claim, 1_000_000)

	if result.Ceiling != 950_000 {
		t.Errorf("expected 950000 (biological stratum), got %d", result.Ceiling)
	}
}

func TestInferStratum(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"mathematics", "fundamental"},
		{"physics", "physical"},
		{"chemistry", "chemical"},
		{"biology", "biological"},
		{"cognitive_science", "cognitive"},
		{"sociology", "social"},
		{"computer_science", "technological"},
		{"unknown_domain", "technological"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := inferStratum(tt.domain)
			if got != tt.expected {
				t.Errorf("inferStratum(%s) = %s, want %s", tt.domain, got, tt.expected)
			}
		})
	}
}
