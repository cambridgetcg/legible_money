package evaluation

// categoryBaseConfidence maps epistemic categories to base confidence values
// reflecting their inherent verifiability. Higher = more verifiable.
var categoryBaseConfidence = map[string]uint64{
	"protocol":      800_000, // blockchain-verifiable, highest innate certainty
	"analytic":      700_000, // truth-table decidable
	"formal":        700_000, // proof-based
	"computational": 650_000, // deterministic computation
	"replicated":    550_000, // reproducible observation
	"empirical":     500_000, // requires strong evidence
	"historical":    450_000, // requires document/attestation
	"social":        400_000, // trust-weighted, inherently soft
	"predictive":    350_000, // inherently uncertain
}

const defaultCategoryBase = 400_000 // conservative default for unknown categories

// evaluateCategory returns a StageResult with the base confidence for the
// claim's epistemic category. This is Stage 1 of the evaluation pipeline.
func evaluateCategory(category string) StageResult {
	base, ok := categoryBaseConfidence[category]
	if !ok {
		base = defaultCategoryBase
	}

	return StageResult{
		Name:          "category_base",
		Adjustment:    int64(base),
		Justification: "base confidence for category: " + category,
	}
}
