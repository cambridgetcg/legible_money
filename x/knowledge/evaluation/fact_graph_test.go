package evaluation

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func makeFacts(facts map[string]types.Fact) func(sdk.Context, string) (types.Fact, bool) {
	return func(_ sdk.Context, id string) (types.Fact, bool) {
		f, ok := facts[id]
		return f, ok
	}
}

func TestEvaluateFactGraph_NoReferences(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{}
	claim := types.Claim{Id: "c1"}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0, got %d", result.Adjustment)
	}
}

func TestEvaluateFactGraph_VerifiedReferences(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Confidence: 800_000},
		"f2": {Id: "f2", Status: types.FactStatusActive, Confidence: 700_000},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{
		Id:         "c1",
		References: []string{"f1", "f2"},
	}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// 2 verified facts × 30,000 = 60,000
	if result.Adjustment != 60_000 {
		t.Errorf("expected 60000, got %d", result.Adjustment)
	}
}

func TestEvaluateFactGraph_SupportCapped(t *testing.T) {
	facts := map[string]types.Fact{}
	for i := 0; i < 10; i++ {
		id := "f" + string(rune('0'+i))
		facts[id] = types.Fact{Id: id, Status: types.FactStatusVerified}
	}

	ctx := mockCtx()
	refs := make([]string, 10)
	for i := range refs {
		refs[i] = "f" + string(rune('0'+i))
	}
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", References: refs}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// 10 × 30,000 = 300,000 but capped at 150,000
	if result.Adjustment != int64(maxSupportBonus) {
		t.Errorf("expected %d, got %d", maxSupportBonus, result.Adjustment)
	}
}

func TestEvaluateFactGraph_DisprovenReferences(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusDisproven},
		"f2": {Id: "f2", Status: types.FactStatusRevoked},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", References: []string{"f1", "f2"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// 2 × -50,000 = -100,000
	if result.Adjustment != -100_000 {
		t.Errorf("expected -100000, got %d", result.Adjustment)
	}
}

func TestEvaluateFactGraph_BrokenReferences(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact: func(_ sdk.Context, _ string) (types.Fact, bool) { return types.Fact{}, false },
	}
	claim := types.Claim{Id: "c1", References: []string{"nonexistent"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	if result.Adjustment != -int64(brokenRefPenalty) {
		t.Errorf("expected %d, got %d", -int64(brokenRefPenalty), result.Adjustment)
	}
}

func TestEvaluateFactGraph_HighConfidenceContradiction(t *testing.T) {
	facts := map[string]types.Fact{
		"f1":     {Id: "f1", Status: types.FactStatusVerified, Confidence: 800_000},
		"contra": {Id: "contra", Status: types.FactStatusActive, Confidence: 800_000},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact: makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, factId string) []types.FactReference {
			if factId == "f1" {
				return []types.FactReference{
					{FromFactId: "contra", ToFactId: "f1", ReferenceType: "contradicts"},
				}
			}
			return nil
		},
	}
	claim := types.Claim{Id: "c1", References: []string{"f1"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// +30,000 support - 50,000 contradiction = -20,000
	if result.Adjustment != -20_000 {
		t.Errorf("expected -20000, got %d", result.Adjustment)
	}
}

// --- Domain Coherence Tests (AH-3) ---

func TestDomainCoherence_SameDomain(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: "physics"},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "physics", References: []string{"f1"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// Same domain → no incoherence penalty, just +30,000 support bonus
	if result.Adjustment != 30_000 {
		t.Errorf("expected 30000 (no domain penalty), got %d", result.Adjustment)
	}
}

func TestDomainCoherence_SameStratum(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: "astronomy"}, // physical stratum
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "physics", References: []string{"f1"}} // physical stratum

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// Same stratum penalty: +30,000 support - 5,000 same-stratum = 25,000
	if result.Adjustment != 25_000 {
		t.Errorf("expected 25000, got %d", result.Adjustment)
	}
}

func TestDomainCoherence_DifferentStratum(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: "sociology"}, // social stratum
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "physics", References: []string{"f1"}} // physical stratum

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// Different stratum penalty: +30,000 support - 20,000 diff-stratum = 10,000
	if result.Adjustment != 10_000 {
		t.Errorf("expected 10000, got %d", result.Adjustment)
	}
}

func TestDomainCoherence_MultipleIncoherent(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: "sociology"},  // social
		"f2": {Id: "f2", Status: types.FactStatusVerified, Domain: "economics"},  // social
		"f3": {Id: "f3", Status: types.FactStatusVerified, Domain: "psychology"}, // cognitive
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "physics", References: []string{"f1", "f2", "f3"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// 3 × 30,000 support = 90,000
	// f1: different stratum (social vs physical) = -20,000
	// f2: different stratum (social vs physical) = -20,000
	// f3: different stratum (cognitive vs physical) = -20,000
	// Total: 90,000 - 60,000 = 30,000
	if result.Adjustment != 30_000 {
		t.Errorf("expected 30000, got %d", result.Adjustment)
	}
}

func TestDomainCoherence_Capped(t *testing.T) {
	facts := map[string]types.Fact{}
	refs := []string{}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("f%d", i)
		facts[id] = types.Fact{Id: id, Status: types.FactStatusVerified, Domain: "sociology"}
		refs = append(refs, id)
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "mathematics", References: refs}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// 10 × 30,000 = 300,000 capped at 150,000 (support)
	// 10 × 20,000 = 200,000 capped at 100,000 (incoherence)
	// Total: 150,000 - 100,000 = 50,000
	if result.Adjustment != 50_000 {
		t.Errorf("expected 50000, got %d", result.Adjustment)
	}
}

func TestDomainCoherence_EmptyDomain_NoCheck(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: "sociology"},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "", References: []string{"f1"}} // empty domain → skip check

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// No domain coherence check when claim domain is empty
	if result.Adjustment != 30_000 {
		t.Errorf("expected 30000 (no domain check), got %d", result.Adjustment)
	}
}

func TestDomainCoherence_RefEmptyDomain_Skipped(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Domain: ""},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetFact:               makeFacts(facts),
		GetContradictingFacts: func(_ sdk.Context, _ string) []types.FactReference { return nil },
	}
	claim := types.Claim{Id: "c1", Domain: "physics", References: []string{"f1"}}

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// Ref with empty domain → no penalty
	if result.Adjustment != 30_000 {
		t.Errorf("expected 30000 (ref empty domain skipped), got %d", result.Adjustment)
	}
}

func TestDetectCycle_NoCycle(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", References: []string{"f2"}},
		"f2": {Id: "f2", References: nil},
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{GetFact: makeFacts(facts)}

	if detectCycle(ctx, evalCtx, []string{"f1"}) {
		t.Error("expected no cycle")
	}
}

func TestDetectCycle_WithCycle(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", References: []string{"f2"}},
		"f2": {Id: "f2", References: []string{"f1"}}, // cycle back to f1
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{GetFact: makeFacts(facts)}

	if !detectCycle(ctx, evalCtx, []string{"f1"}) {
		t.Error("expected cycle detection")
	}
}

func TestDetectCycle_BeyondMaxDepth(t *testing.T) {
	// ZD-K01: maxBFSDepth increased from 5 to 10.
	// Create a chain longer than maxBFSDepth=10 that loops back.
	facts := map[string]types.Fact{
		"f1":  {Id: "f1", References: []string{"f2"}},
		"f2":  {Id: "f2", References: []string{"f3"}},
		"f3":  {Id: "f3", References: []string{"f4"}},
		"f4":  {Id: "f4", References: []string{"f5"}},
		"f5":  {Id: "f5", References: []string{"f6"}},
		"f6":  {Id: "f6", References: []string{"f7"}},
		"f7":  {Id: "f7", References: []string{"f8"}},
		"f8":  {Id: "f8", References: []string{"f9"}},
		"f9":  {Id: "f9", References: []string{"f10"}},
		"f10": {Id: "f10", References: []string{"f11"}},
		"f11": {Id: "f11", References: []string{"f1"}}, // cycle at depth 11 > maxBFSDepth=10
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{GetFact: makeFacts(facts)}

	// Should NOT detect the cycle since it's beyond maxBFSDepth
	if detectCycle(ctx, evalCtx, []string{"f1"}) {
		t.Error("should not detect cycle beyond max depth")
	}
}

func TestDetectCycle_SelfReference(t *testing.T) {
	facts := map[string]types.Fact{
		"f1": {Id: "f1", References: []string{"f1"}}, // self-loop
	}
	ctx := mockCtx()
	evalCtx := &EvaluationContext{GetFact: makeFacts(facts)}

	// BFS starts at f1, visits f1's references which include f1 (already visited + is original)
	if !detectCycle(ctx, evalCtx, []string{"f1"}) {
		t.Error("expected self-reference cycle detection")
	}
}
