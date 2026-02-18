package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// mockSDKContext returns a minimal SDK context for testing
func mockSDKContext() sdk.Context {
	return sdk.Context{}
}

func TestEvaluateCounterFacts_NoLookup(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{ExtendedParams: types.DefaultExtendedParams()}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Name != "counter_facts" {
		t.Errorf("expected stage name counter_facts, got %s", result.Name)
	}
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment without lookup, got %d", result.Adjustment)
	}
}

func TestEvaluateCounterFacts_NoReferences(t *testing.T) {
	ctx := mockSDKContext()
	called := false
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			called = true
			return nil
		},
	}
	claim := types.Claim{
		References: []string{},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if called {
		t.Error("should not call GetCounterFactsByFact with no references")
	}
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment, got %d", result.Adjustment)
	}
}

func TestEvaluateCounterFacts_HighConfidenceCounterFact(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, factId string) []types.CounterFact {
			if factId == "fact-1" {
				return []types.CounterFact{
					{
						Id:         "cfact-1",
						Confidence: 800_000, // High
						Status:     types.CounterFactStatusVerified,
					},
				}
			}
			return nil
		},
	}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Adjustment != -int64(types.CounterFactHighPenalty) {
		t.Errorf("expected -%d penalty for high confidence counter-fact, got %d",
			types.CounterFactHighPenalty, result.Adjustment)
	}
}

func TestEvaluateCounterFacts_MediumConfidenceCounterFact(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			return []types.CounterFact{
				{
					Id:         "cfact-1",
					Confidence: 500_000, // Medium
					Status:     types.CounterFactStatusVerified,
				},
			}
		},
	}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Adjustment != -int64(types.CounterFactMediumPenalty) {
		t.Errorf("expected -%d penalty, got %d", types.CounterFactMediumPenalty, result.Adjustment)
	}
}

func TestEvaluateCounterFacts_LowConfidenceCounterFact(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			return []types.CounterFact{
				{
					Id:         "cfact-1",
					Confidence: 200_000, // Low
					Status:     types.CounterFactStatusVerified,
				},
			}
		},
	}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Adjustment != -int64(types.CounterFactLowPenalty) {
		t.Errorf("expected -%d penalty, got %d", types.CounterFactLowPenalty, result.Adjustment)
	}
}

func TestEvaluateCounterFacts_UnverifiedIgnored(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			return []types.CounterFact{
				{
					Id:         "cfact-1",
					Confidence: 900_000,
					Status:     types.CounterFactStatusPending, // Not verified
				},
			}
		},
	}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 penalty for unverified counter-fact, got %d", result.Adjustment)
	}
}

func TestEvaluateCounterFacts_PenaltyCapped(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			// Return many high-confidence counter-facts
			cfs := make([]types.CounterFact, 10)
			for i := range cfs {
				cfs[i] = types.CounterFact{
					Id:         "cfact-" + string(rune('0'+i)),
					Confidence: 900_000,
					Status:     types.CounterFactStatusVerified,
				}
			}
			return cfs
		},
	}
	// 5 references × 10 counter-facts each = 50 × 80,000 = 4,000,000 uncapped
	claim := types.Claim{
		References: []string{"f1", "f2", "f3", "f4", "f5"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.Adjustment != -int64(types.MaxCounterFactPenalty) {
		t.Errorf("expected penalty capped at -%d, got %d",
			types.MaxCounterFactPenalty, result.Adjustment)
	}
}

func TestEvaluateCounterFacts_DisprovenRefForcesReject(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, _ string) []types.CounterFact {
			return []types.CounterFact{
				{
					Id:         "cfact-1",
					Confidence: 900_000,
					Status:     types.CounterFactStatusVerified,
				},
			}
		},
		GetBidirectionalConfidence: func(_ sdk.Context, factId string) types.BidirectionalConfidence {
			return types.BidirectionalConfidence{
				FactId:            factId,
				TruthConfidence:   100_000,
				FalsityConfidence: 900_000,
				NetConfidence:     -800_000,
				CounterFactCount:  1,
				Status:            "disproven",
			}
		},
	}
	claim := types.Claim{
		References: []string{"fact-1"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	if result.ForceVerdict != "reject" {
		t.Errorf("expected forced reject for disproven ref, got %q", result.ForceVerdict)
	}
}

func TestEvaluateCounterFacts_MultipleRefs(t *testing.T) {
	ctx := mockSDKContext()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetCounterFactsByFact: func(_ sdk.Context, factId string) []types.CounterFact {
			switch factId {
			case "fact-1":
				return []types.CounterFact{
					{Confidence: 800_000, Status: types.CounterFactStatusVerified}, // High: -80k
				}
			case "fact-2":
				return []types.CounterFact{
					{Confidence: 500_000, Status: types.CounterFactStatusVerified}, // Medium: -40k
				}
			}
			return nil
		},
	}
	claim := types.Claim{
		References: []string{"fact-1", "fact-2"},
	}
	result := evaluateCounterFacts(ctx, evalCtx, claim)
	expectedPenalty := int64(types.CounterFactHighPenalty + types.CounterFactMediumPenalty)
	if result.Adjustment != -expectedPenalty {
		t.Errorf("expected -%d total penalty, got %d", expectedPenalty, result.Adjustment)
	}
}

func TestComputeBidirectionalStatus(t *testing.T) {
	tests := []struct {
		net      int64
		expected string
	}{
		{800_000, "established"},
		{-800_000, "disproven"},
		{0, "uncertain"},
		{100_000, "uncertain"},
		{-100_000, "uncertain"},
		{400_000, "contested"},
		{-400_000, "contested"},
	}
	for _, tt := range tests {
		result := types.ComputeBidirectionalStatus(tt.net)
		if result != tt.expected {
			t.Errorf("ComputeBidirectionalStatus(%d) = %q, want %q", tt.net, result, tt.expected)
		}
	}
}
