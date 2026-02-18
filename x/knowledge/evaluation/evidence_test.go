package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func mockCtx() sdk.Context {
	return sdk.Context{}
}

func TestEvaluateEvidence_NoEvidence_RequiringCategory(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return nil
		},
	}
	claim := types.Claim{
		Id:                "claim-1",
		EpistemicCategory: "empirical",
		References:        nil, // no references either
	}

	result := evaluateEvidence(ctx, evalCtx, claim)
	if result.Adjustment != -int64(noEvidencePenalty) {
		t.Errorf("expected penalty %d, got %d", -int64(noEvidencePenalty), result.Adjustment)
	}
}

func TestEvaluateEvidence_NoEvidence_NonRequiringCategory(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return nil
		},
	}
	claim := types.Claim{
		Id:                "claim-1",
		EpistemicCategory: "analytic", // doesn't require evidence
	}

	result := evaluateEvidence(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment for non-requiring category, got %d", result.Adjustment)
	}
}

func TestEvaluateEvidence_WithOnChainEvidence(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{
					Id:           "ev-1",
					EvidenceType: "on_chain",
					Status:       "verified",
					Reliability:  900_000, // 90%
				},
			}
		},
	}
	claim := types.Claim{Id: "claim-1", EpistemicCategory: "protocol"}

	result := evaluateEvidence(ctx, evalCtx, claim)

	// on_chain strength = 1,000,000, reliability = 900,000
	// score = 1,000,000 * 900,000 / 1,000,000 = 900,000
	// avg = 900,000 / 1 = 900,000, capped at MaxEvidenceBonus = 150,000
	expected := int64(types.MaxEvidenceBonus)
	if result.Adjustment != expected {
		t.Errorf("expected %d, got %d", expected, result.Adjustment)
	}
}

func TestEvaluateEvidence_RejectedEvidenceSkipped(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", EvidenceType: "on_chain", Status: "rejected", Reliability: 900_000},
				{Id: "ev-2", EvidenceType: "on_chain", Status: "expired", Reliability: 800_000},
			}
		},
	}
	claim := types.Claim{Id: "claim-1", EpistemicCategory: "protocol"}

	result := evaluateEvidence(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 for all rejected/expired, got %d", result.Adjustment)
	}
}

func TestEvaluateEvidence_UnauditedEvidence(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", EvidenceType: "document", Status: "submitted", Reliability: 0},
			}
		},
	}
	claim := types.Claim{Id: "claim-1", EpistemicCategory: "historical"}

	result := evaluateEvidence(ctx, evalCtx, claim)

	// document strength = 600,000, base reliability = 100,000
	// score = 600,000 * 100,000 / 1,000,000 = 60,000
	if result.Adjustment != 60_000 {
		t.Errorf("expected 60000, got %d", result.Adjustment)
	}
}

func TestEvaluateEvidence_NilEvidenceKeeper(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams:       types.DefaultExtendedParams(),
		GetEvidenceByClaimId: nil, // no evidence keeper
	}
	claim := types.Claim{
		Id:                "claim-1",
		EpistemicCategory: "empirical",
		References:        []string{"ref-1"},
	}

	result := evaluateEvidence(ctx, evalCtx, claim)
	// Evidence penalty applies regardless of references for evidence-requiring categories
	if result.Adjustment != -int64(noEvidencePenalty) {
		t.Errorf("expected -%d penalty for empirical with no evidence, got %d", noEvidencePenalty, result.Adjustment)
	}
}

func TestEvaluateEvidence_MultipleEvidences(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 800_000},
				{Id: "ev-2", EvidenceType: "attestation", Status: "submitted", Reliability: 500_000},
			}
		},
	}
	claim := types.Claim{Id: "claim-1", EpistemicCategory: "empirical"}

	result := evaluateEvidence(ctx, evalCtx, claim)

	// ev-1: 1,000,000 * 800,000 / 1,000,000 = 800,000
	// ev-2: 500,000 * 500,000 / 1,000,000 = 250,000
	// avg = (800,000 + 250,000) / 2 = 525,000 → capped at 150,000
	if result.Adjustment != int64(types.MaxEvidenceBonus) {
		t.Errorf("expected %d (capped), got %d", types.MaxEvidenceBonus, result.Adjustment)
	}
}

func TestEvaluateEvidence_UnknownType(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", EvidenceType: "mystery_type", Status: "verified", Reliability: 1_000_000},
			}
		},
	}
	claim := types.Claim{Id: "claim-1", EpistemicCategory: "protocol"}

	result := evaluateEvidence(ctx, evalCtx, claim)

	// unknown type strength = 400,000, reliability = 1,000,000
	// score = 400,000 * 1,000,000 / 1,000,000 = 400,000 → capped at 150,000
	if result.Adjustment != int64(types.MaxEvidenceBonus) {
		t.Errorf("expected %d, got %d", types.MaxEvidenceBonus, result.Adjustment)
	}
}
