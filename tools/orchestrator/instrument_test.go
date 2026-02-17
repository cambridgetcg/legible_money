package orchestrator

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// --- StringInstrument Tests ---

func TestStringInstrumentCategoryConfidence(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-protocol",
		Category: "protocol",
		Domain:   "general",
	})
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-predictive",
		Category: "predictive",
		Domain:   "general",
	})

	inst := NewStringInstrument(StrategyConservative, nil, resolver)
	ctx := context.Background()

	// Protocol: base 800k × 0.8 (conservative) = 640k → True
	verdict, conf, err := inst.Evaluate(ctx, "claim-protocol")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected True for protocol, got %s", verdict)
	}
	if conf != 640_000 {
		t.Fatalf("expected 640000, got %d", conf)
	}

	// Predictive: base 350k × 0.8 = 280k → False
	verdict, conf, err = inst.Evaluate(ctx, "claim-predictive")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict != agentsdk.VerdictFalse {
		t.Fatalf("expected False for predictive (conservative), got %s", verdict)
	}
	if conf != 280_000 {
		t.Fatalf("expected 280000, got %d", conf)
	}
}

func TestStringInstrumentStrategyAggressive(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-social",
		Category: "social",
		Domain:   "general",
	})

	inst := NewStringInstrument(StrategyAggressive, nil, resolver)
	ctx := context.Background()

	// Social: base 400k × 1.2 (aggressive) = 480k → Uncertain
	verdict, conf, _ := inst.Evaluate(ctx, "claim-social")
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected Uncertain for social (aggressive), got %s", verdict)
	}
	if conf != 480_000 {
		t.Fatalf("expected 480000, got %d", conf)
	}
}

func TestStringInstrumentStrategySpecialist(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-math",
		Category: "formal",
		Domain:   "mathematics",
	})
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-physics",
		Category: "empirical",
		Domain:   "physics",
	})

	inst := NewStringInstrument(StrategySpecialist, []string{"mathematics"}, resolver)
	ctx := context.Background()

	// Math (specialist domain): base 700k × 1.1 = 770k → True
	_, conf, _ := inst.Evaluate(ctx, "claim-math")
	if conf != 770_000 {
		t.Fatalf("expected 770000 for specialist domain, got %d", conf)
	}

	// Physics (non-specialist): base 500k × 1.0 = 500k → True (at threshold)
	_, conf, _ = inst.Evaluate(ctx, "claim-physics")
	if conf != 500_000 {
		t.Fatalf("expected 500000 for non-specialist, got %d", conf)
	}
}

func TestStringInstrumentCacheDeterminism(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-1",
		Category: "formal",
		Domain:   "general",
	})

	inst := NewStringInstrument(StrategyConservative, nil, resolver)
	ctx := context.Background()

	// First call
	v1, c1, _ := inst.Evaluate(ctx, "claim-1")

	// Second call should return same result from cache
	v2, c2, _ := inst.Evaluate(ctx, "claim-1")

	if v1 != v2 || c1 != c2 {
		t.Fatalf("cache not deterministic: (%s,%d) vs (%s,%d)", v1, c1, v2, c2)
	}

	if inst.CacheSize() != 1 {
		t.Fatalf("expected 1 cached entry, got %d", inst.CacheSize())
	}

	inst.ClearCache()
	if inst.CacheSize() != 0 {
		t.Fatal("cache not cleared")
	}
}

func TestStringInstrumentUnknownCategory(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-unknown",
		Category: "alien_science",
		Domain:   "general",
	})

	inst := NewStringInstrument(StrategyConservative, nil, resolver)
	ctx := context.Background()

	// Unknown category: default base 400k × 0.8 = 320k → Uncertain
	verdict, conf, _ := inst.Evaluate(ctx, "claim-unknown")
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected Uncertain for unknown category, got %s", verdict)
	}
	if conf != 320_000 {
		t.Fatalf("expected 320000 for unknown category, got %d", conf)
	}
}

func TestStringInstrumentUnresolvable(t *testing.T) {
	resolver := NewMockClaimResolver() // Empty — no claims

	inst := NewStringInstrument(StrategyConservative, nil, resolver)
	ctx := context.Background()

	// Unresolvable claim → Uncertain with default confidence
	verdict, _, _ := inst.Evaluate(ctx, "nonexistent-claim")
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected Uncertain for unresolvable claim, got %s", verdict)
	}
}

func TestStringInstrumentAggressiveCap(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "claim-max",
		Category: "protocol",
		Domain:   "general",
	})

	inst := NewStringInstrument(StrategyAggressive, nil, resolver)
	ctx := context.Background()

	// Protocol: 800k × 1.2 = 960k (below cap)
	_, conf, _ := inst.Evaluate(ctx, "claim-max")
	if conf != 960_000 {
		t.Fatalf("expected 960000, got %d", conf)
	}
	if conf > 1_000_000 {
		t.Fatal("confidence exceeded maximum")
	}
}

// --- BrassInstrument Tests ---

func TestBrassInstrumentTemplateClaim(t *testing.T) {
	strategy := NewTemplateClaimStrategy(map[string][]ClaimTemplate{
		"mathematics": {
			{Text: "The fundamental theorem of calculus connects differentiation and integration", Category: "formal", Stake: 5_000_000},
			{Text: "Euler's identity relates five fundamental mathematical constants", Category: "analytic", Stake: 3_000_000},
		},
	})

	inst := NewBrassInstrument(strategy, []string{"mathematics"}, 1_000_000)
	ctx := context.Background()

	params := map[string]string{"domain": "mathematics"}
	result, err := inst.PrepareAction(ctx, nil, ActionSubmitClaim, params)
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["text"] == "" {
		t.Fatal("expected claim text to be populated")
	}
	if result["category"] != "formal" {
		t.Fatalf("expected formal category, got %s", result["category"])
	}

	// Second call should cycle to next template
	result2, _ := inst.PrepareAction(ctx, nil, ActionSubmitClaim, params)
	if result2["category"] != "analytic" {
		t.Fatalf("expected analytic category (cycled), got %s", result2["category"])
	}
}

func TestBrassInstrumentMinStake(t *testing.T) {
	strategy := NewTemplateClaimStrategy(map[string][]ClaimTemplate{
		"general": {
			{Text: "A simple test claim for min stake testing purposes", Category: "computational", Stake: 100}, // Below min
		},
	})

	inst := NewBrassInstrument(strategy, []string{"general"}, 1_000_000)
	ctx := context.Background()

	result, err := inst.PrepareAction(ctx, nil, ActionSubmitClaim, map[string]string{"domain": "general"})
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["stake"] != "1000000" {
		t.Fatalf("expected min stake 1000000, got %s", result["stake"])
	}
}

func TestBrassInstrumentDomainFiltering(t *testing.T) {
	strategy := NewTemplateClaimStrategy(map[string][]ClaimTemplate{
		"physics": {
			{Text: "Mass-energy equivalence is described by E equals mc squared", Category: "empirical", Stake: 5_000_000},
		},
	})

	inst := NewBrassInstrument(strategy, []string{"physics"}, 1_000_000)
	ctx := context.Background()

	// Request for domain with no templates
	_, err := inst.PrepareAction(ctx, nil, ActionSubmitClaim, map[string]string{"domain": "mathematics"})
	if err == nil {
		t.Fatal("expected error for domain with no templates")
	}
}

func TestBrassInstrumentPassthrough(t *testing.T) {
	inst := NewBrassInstrument(nil, nil, 0)
	ctx := context.Background()

	// Non-claim action should pass through
	params := map[string]string{"test": "value"}
	result, err := inst.PrepareAction(ctx, nil, ActionHeartbeat, params)
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["test"] != "value" {
		t.Fatal("expected passthrough for non-claim action")
	}

	// Claim with pre-existing text should pass through
	params = map[string]string{"text": "already set", "domain": "general"}
	result, err = inst.PrepareAction(ctx, nil, ActionSubmitClaim, params)
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["text"] != "already set" {
		t.Fatal("expected passthrough when text already set")
	}
}

// --- PercussionInstrument Tests ---

func TestPercussionDetectorScoring(t *testing.T) {
	detector := NewDefaultChallengeDetector(100_000)

	facts := []agentsdk.FactInfo{
		{ID: "fact-high", Confidence: 800_000, Category: "formal"},     // Above floor — skip
		{ID: "fact-low", Confidence: 300_000, Category: "predictive"},  // Below floor
		{ID: "fact-mid", Confidence: 500_000, Category: "empirical"},   // Below floor
	}

	candidates, err := detector.DetectChallenges(context.Background(), facts)
	if err != nil {
		t.Fatalf("DetectChallenges: %v", err)
	}

	// Should have 2 candidates (fact-high is above floor)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// fact-low should score higher (lower confidence = higher score)
	if candidates[0].FactID != "fact-low" {
		t.Fatalf("expected fact-low first (highest score), got %s", candidates[0].FactID)
	}
}

func TestPercussionDetectorSortTiebreaker(t *testing.T) {
	detector := &DefaultChallengeDetector{
		ConfidenceFloor: 600_000,
		CategoryDecay:   map[string]uint64{},
		CurrentBlock:    100_000,
	}

	facts := []agentsdk.FactInfo{
		{ID: "fact-b", Confidence: 400_000, Category: "unknown"},
		{ID: "fact-a", Confidence: 400_000, Category: "unknown"},
	}

	candidates, _ := detector.DetectChallenges(context.Background(), facts)
	if len(candidates) != 2 {
		t.Fatalf("expected 2, got %d", len(candidates))
	}

	// Same score — should sort by FactID lexicographically
	if candidates[0].FactID != "fact-a" {
		t.Fatalf("expected fact-a first (lexicographic tiebreak), got %s", candidates[0].FactID)
	}
}

func TestPercussionInstrumentPrepare(t *testing.T) {
	scanner := NewMockFactScanner()
	scanner.AddFact("general", agentsdk.FactInfo{
		ID:         "weak-fact",
		Confidence: 300_000,
		Category:   "social",
		Domain:     "general",
	})

	detector := NewDefaultChallengeDetector(100_000)
	inst := NewPercussionInstrument(detector, scanner, []string{"general"})
	ctx := context.Background()

	result, err := inst.PrepareAction(ctx, nil, ActionChallenge, map[string]string{})
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["fact_id"] != "weak-fact" {
		t.Fatalf("expected weak-fact, got %s", result["fact_id"])
	}
	if result["stake"] != "11000000" {
		t.Fatalf("expected min stake, got %s", result["stake"])
	}
}

func TestPercussionInstrumentNoCandidates(t *testing.T) {
	scanner := NewMockFactScanner()
	// All facts above floor
	scanner.AddFact("general", agentsdk.FactInfo{
		ID:         "strong-fact",
		Confidence: 900_000,
		Category:   "protocol",
		Domain:     "general",
	})

	detector := NewDefaultChallengeDetector(100_000)
	inst := NewPercussionInstrument(detector, scanner, []string{"general"})

	_, err := inst.PrepareAction(context.Background(), nil, ActionChallenge, map[string]string{})
	if err == nil {
		t.Fatal("expected error when no challenge candidates found")
	}
}

// --- WindInstrument Tests ---

func TestWindInstrumentDefenseStrategy(t *testing.T) {
	inst := NewWindInstrument(DefenseActive, 10_000)
	ctx := context.Background()

	result, err := inst.PrepareAction(ctx, nil, ActionCreateHome, map[string]string{
		"name": "test-home",
	})
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["defense_strategy"] != "active" {
		t.Fatalf("expected active strategy, got %s", result["defense_strategy"])
	}
	if result["deadman_threshold"] != "10000" {
		t.Fatalf("expected 10000 threshold, got %s", result["deadman_threshold"])
	}
	if result["auto_defend"] != "true" {
		t.Fatalf("expected auto_defend=true for active strategy")
	}
}

func TestWindInstrumentPassiveStrategy(t *testing.T) {
	inst := NewWindInstrument(DefensePassive, 0)
	ctx := context.Background()

	result, err := inst.PrepareAction(ctx, nil, ActionCreateHome, map[string]string{})
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["defense_strategy"] != "passive" {
		t.Fatalf("expected passive strategy, got %s", result["defense_strategy"])
	}
	if result["auto_defend"] == "true" {
		t.Fatal("passive strategy should not set auto_defend")
	}
}

func TestWindInstrumentSpendingLimits(t *testing.T) {
	inst := NewWindInstrument(DefenseAggressive, 5_000)
	inst.SetSpendingLimit("session", 50_000_000)
	inst.SetSpendingLimit("operator", 100_000_000)

	actions := inst.GenerateGuardianActions("home-123")

	// Should have: configure_guardian + 2 spending limits = 3 actions
	if len(actions) != 3 {
		t.Fatalf("expected 3 guardian actions, got %d", len(actions))
	}

	// First should be configure_guardian
	if actions[0].Type != "configure_guardian" {
		t.Fatalf("expected configure_guardian, got %s", actions[0].Type)
	}
}

func TestWindInstrumentNonHomePassthrough(t *testing.T) {
	inst := NewWindInstrument(DefenseActive, 10_000)
	ctx := context.Background()

	params := map[string]string{"test": "value"}
	result, err := inst.PrepareAction(ctx, nil, ActionHeartbeat, params)
	if err != nil {
		t.Fatalf("PrepareAction: %v", err)
	}
	if result["test"] != "value" {
		t.Fatal("expected passthrough for non-home action")
	}
}

// --- MockClaimResolver Tests ---

func TestMockClaimResolver(t *testing.T) {
	resolver := NewMockClaimResolver()
	resolver.AddClaim(agentsdk.ClaimInfo{
		ID:       "test-claim",
		Category: "formal",
		Domain:   "math",
	})

	info, err := resolver.ResolveClaim(context.Background(), "test-claim")
	if err != nil {
		t.Fatalf("ResolveClaim: %v", err)
	}
	if info.Category != "formal" {
		t.Fatalf("expected formal, got %s", info.Category)
	}

	_, err = resolver.ResolveClaim(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing claim")
	}
}

func TestMockFactScanner(t *testing.T) {
	scanner := NewMockFactScanner()
	scanner.AddFact("physics", agentsdk.FactInfo{ID: "f1", Confidence: 500_000})
	scanner.AddFact("physics", agentsdk.FactInfo{ID: "f2", Confidence: 300_000})

	facts, err := scanner.ScanFacts(context.Background(), "physics")
	if err != nil {
		t.Fatalf("ScanFacts: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
}
