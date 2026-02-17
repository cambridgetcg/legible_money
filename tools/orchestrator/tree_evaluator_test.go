package orchestrator

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

func makeTestTreeEvaluator(kctx *KnowledgeContext) (*TreeAwareEvaluator, *MockClaimResolver) {
	resolver := NewMockClaimResolver()
	inner := NewStringInstrument(StrategyConservative, nil, resolver)

	mockCtxProvider := NewMockKnowledgeContextProvider()
	if kctx != nil && kctx.Claim != nil {
		mockCtxProvider.AddContext(kctx.Claim.ID, kctx)
	}

	return NewTreeAwareEvaluator(TreeAwareEvaluatorConfig{
		Inner:       inner,
		CtxProvider: mockCtxProvider,
		Navigator:   NewMockKnowledgeNavigator(),
	}), resolver
}

func TestTreeEvaluatorBaseEvaluation(t *testing.T) {
	// No knowledge context — should fall back to StringInstrument base
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-1", Category: "protocol", Domain: "general",
	})

	inner := NewStringInstrument(StrategyConservative, nil, resolver)
	eval := NewTreeAwareEvaluator(TreeAwareEvaluatorConfig{
		Inner: inner,
		// No ctxProvider — graceful degradation
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Protocol base 800k × 0.8 (conservative) = 640k → True
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected True, got %s", verdict)
	}
	if conf != 640_000 {
		t.Fatalf("expected 640000 (base only), got %d", conf)
	}
}

func TestTreeEvaluatorStratumCeiling(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-social", Category: "protocol", Domain: "social",
		},
		StratumCeiling: 500_000, // artificially low ceiling
		LogicZoneCap:   1_000_000,
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-social", Category: "protocol", Domain: "social",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-social")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 800k × 0.8 = 640k, but stratum ceiling 500k should cap it
	if conf > 500_000 {
		t.Fatalf("expected conf <= 500000 (stratum ceiling), got %d", conf)
	}
}

func TestTreeEvaluatorLogicZoneCap(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-logic", Category: "protocol", Domain: "general",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   450_000, // artificially low logic zone
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-logic", Category: "protocol", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-logic")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 640k, logic zone cap 450k should cap it
	if conf > 450_000 {
		t.Fatalf("expected conf <= 450000 (logic zone cap), got %d", conf)
	}
}

func TestTreeEvaluatorReferenceChainBonus(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-refs", Category: "empirical", Domain: "general",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		SupportingFacts: []agentsdk.FactInfo{
			{ID: "f1", Confidence: 800_000},
			{ID: "f2", Confidence: 700_000},
			{ID: "f3", Confidence: 600_000},
		},
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-refs", Category: "empirical", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-refs")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 500k × 0.8 = 400k, + 3 × 500 = 1500 BPS bonus = 401500
	// Expected: 400000 + 1500 = 401500
	if conf < 400_000 {
		t.Fatalf("expected conf >= 400000 with reference bonus, got %d", conf)
	}
	if conf <= 400_000 {
		t.Fatalf("expected reference chain bonus to increase confidence above 400000, got %d", conf)
	}
}

func TestTreeEvaluatorReferenceChainBonusCapped(t *testing.T) {
	// 5 supporting facts → 5 × 500 = 2500, capped at 2000
	supporting := make([]agentsdk.FactInfo, 5)
	for i := range supporting {
		supporting[i] = agentsdk.FactInfo{ID: "f", Confidence: 800_000}
	}

	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-max-refs", Category: "empirical", Domain: "general",
		},
		StratumCeiling:  1_000_000,
		LogicZoneCap:    1_000_000,
		SupportingFacts: supporting,
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-max-refs", Category: "empirical", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-max-refs")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 400k + capped 2000 = 402000
	if conf != 402_000 {
		t.Fatalf("expected 402000 (capped ref bonus), got %d", conf)
	}
}

func TestTreeEvaluatorCounterFactPenalty(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-countered", Category: "protocol", Domain: "general",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		CounterFacts: []agentsdk.CounterFactInfo{
			{ID: "cf1", NegationType: "contradiction", Confidence: 600_000},
			{ID: "cf2", NegationType: "falsification", Confidence: 700_000},
		},
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-countered", Category: "protocol", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-countered")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 640k, - 2 × 1000 = 2000 penalty = 638000
	if conf != 638_000 {
		t.Fatalf("expected 638000 (counter-fact penalty), got %d", conf)
	}
}

func TestTreeEvaluatorCounterFactPenaltyFloor(t *testing.T) {
	// Many counter-facts should not go below 0
	counterFacts := make([]agentsdk.CounterFactInfo, 100)
	for i := range counterFacts {
		counterFacts[i] = agentsdk.CounterFactInfo{ID: "cf", Confidence: 500_000}
	}

	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-destroyed", Category: "social", Domain: "general",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		CounterFacts:   counterFacts,
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-destroyed", Category: "social", Domain: "general",
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "claim-destroyed")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Should be 0 or very low (False verdict)
	if conf > 10_000 {
		t.Fatalf("expected near-zero confidence with 100 counter-facts, got %d", conf)
	}
	if verdict != agentsdk.VerdictFalse {
		t.Fatalf("expected False with massive counter-facts, got %s", verdict)
	}
}

func TestTreeEvaluatorDomainDensityBonus(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-dense", Category: "empirical", Domain: "physics",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		DomainDensity:  100, // above threshold of 50
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-dense", Category: "empirical", Domain: "physics",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-dense")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 400k + 200 density bonus = 400200
	if conf != 400_200 {
		t.Fatalf("expected 400200 (density bonus), got %d", conf)
	}
}

func TestTreeEvaluatorNoDensityBonusBelowThreshold(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-sparse", Category: "empirical", Domain: "empty",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		DomainDensity:  5, // below threshold of 50
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-sparse", Category: "empirical", Domain: "empty",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-sparse")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 400k, no density bonus
	if conf != 400_000 {
		t.Fatalf("expected 400000 (no density bonus), got %d", conf)
	}
}

func TestTreeEvaluatorCacheDeterminism(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-cache", Category: "formal", Domain: "math",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-cache", Category: "formal", Domain: "math",
	})

	ctx := context.Background()

	// First evaluation
	v1, c1, _ := eval.Evaluate(ctx, "claim-cache")

	// Second evaluation should return from cache
	v2, c2, _ := eval.Evaluate(ctx, "claim-cache")

	if v1 != v2 || c1 != c2 {
		t.Fatalf("cache not deterministic: (%s,%d) vs (%s,%d)", v1, c1, v2, c2)
	}

	if eval.CacheSize() != 1 {
		t.Fatalf("expected 1 cached entry, got %d", eval.CacheSize())
	}

	eval.ClearCache()
	if eval.CacheSize() != 0 {
		t.Fatal("cache not cleared")
	}
}

func TestTreeEvaluatorConfidenceCap(t *testing.T) {
	// Construct a scenario where bonuses would exceed maxConfidenceBps
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-max", Category: "protocol", Domain: "general",
		},
		StratumCeiling: 1_000_000,
		LogicZoneCap:   1_000_000,
		SupportingFacts: []agentsdk.FactInfo{
			{ID: "f1"}, {ID: "f2"}, {ID: "f3"}, {ID: "f4"}, // +2000 (capped)
		},
		DomainDensity: 100, // +200
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-max", Category: "protocol", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-max")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if conf > uint32(maxConfidenceBps) {
		t.Fatalf("confidence %d exceeded max %d", conf, maxConfidenceBps)
	}
}

func TestTreeEvaluatorCombinedAdjustments(t *testing.T) {
	kctx := &KnowledgeContext{
		Claim: &agentsdk.ClaimInfo{
			ID: "claim-combined", Category: "empirical", Domain: "general",
		},
		StratumCeiling: 450_000, // cap at 450k
		LogicZoneCap:   1_000_000,
		SupportingFacts: []agentsdk.FactInfo{
			{ID: "f1"}, {ID: "f2"}, // +1000
		},
		CounterFacts: []agentsdk.CounterFactInfo{
			{ID: "cf1"}, // -1000
		},
		DomainDensity: 100, // +200
	}

	eval, resolver := makeTestTreeEvaluator(kctx)
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID: "claim-combined", Category: "empirical", Domain: "general",
	})

	_, conf, err := eval.Evaluate(context.Background(), "claim-combined")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Base: 400k, stratum caps to 450k (base is lower, no effect)
	// +1000 (2 refs) = 401000, -1000 (1 counter) = 400000, +200 (density) = 400200
	if conf != 400_200 {
		t.Fatalf("expected 400200 for combined adjustments, got %d", conf)
	}
}
