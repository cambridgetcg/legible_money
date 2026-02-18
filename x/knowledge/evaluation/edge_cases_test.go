package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func TestEdgeCase_EmptyClaimText(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{ExtendedParams: types.DefaultExtendedParams()}
	claim := types.Claim{Id: "c1", ClaimText: ""}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "reject" {
		t.Errorf("expected reject for empty claim, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_SelfReference(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{ExtendedParams: types.DefaultExtendedParams()}
	claim := types.Claim{
		Id:          "c1",
		ClaimText:   "some claim",
		ContentHash: "abc123",
		References:  []string{"abc123"}, // content hash == ref ID
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected abstain for self-reference, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_AxiomContradiction(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetFact: func(_ sdk.Context, id string) (types.Fact, bool) {
			if id == "axiom-f1" {
				return types.Fact{
					Id:             "axiom-f1",
					Status:         types.FactStatusDisproven,
					Confidence:     0,
					BaseConfidence: 950_000, // was axiomatic
				}, true
			}
			return types.Fact{}, false
		},
	}
	claim := types.Claim{
		Id:         "c1",
		ClaimText:  "claim referencing disproven axiom",
		References: []string{"axiom-f1"},
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "reject" {
		t.Errorf("expected reject for axiom contradiction, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_AxiomContradiction_LowConfidence(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetFact: func(_ sdk.Context, id string) (types.Fact, bool) {
			if id == "f1" {
				return types.Fact{
					Id:             "f1",
					Status:         types.FactStatusDisproven,
					Confidence:     0,
					BaseConfidence: 500_000, // NOT axiomatic
				}, true
			}
			return types.Fact{}, false
		},
	}
	claim := types.Claim{
		Id:         "c1",
		ClaimText:  "claim referencing disproven non-axiom",
		References: []string{"f1"},
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict == "reject" {
		t.Error("should not force reject for non-axiomatic disproven fact")
	}
}

func TestEdgeCase_NoEvidenceNoRefsRequiringCategory(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return nil // no evidence
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "empirical claim",
		EpistemicCategory: "empirical",
		References:        nil, // no references
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected abstain for no evidence/refs in requiring category, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_NoEvidenceButHasRefs(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return nil
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "empirical claim with refs",
		EpistemicCategory: "empirical",
		References:        []string{"f1"}, // has references but NO evidence
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	// Hard evidence gate: references do NOT substitute for evidence
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected abstain — refs alone insufficient for empirical, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_HasSubstantiveEvidenceNoRefs(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", Status: "verified", EvidenceType: "experiment"}, // substantive
			}
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "empirical claim with substantive evidence",
		EpistemicCategory: "empirical",
		References:        nil,
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "" {
		t.Errorf("should not force verdict with substantive evidence, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_AttestationOnlyInsufficient(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", Status: "verified", EvidenceType: "attestation"}, // NOT substantive
			}
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "empirical claim with only attestation",
		EpistemicCategory: "empirical",
		References:        []string{"f1"},
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected abstain — attestation alone insufficient, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_AnalyticDoesNotRequireEvidence(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return nil
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "bachelors are unmarried",
		EpistemicCategory: "analytic", // NOT evidence-requiring
		References:        nil,
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "" {
		t.Errorf("analytic claims should not require evidence, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_NoEdgeCases(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetFact: func(_ sdk.Context, _ string) (types.Fact, bool) {
			return types.Fact{Id: "f1", Status: types.FactStatusVerified, Confidence: 800_000}, true
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "valid claim",
		EpistemicCategory: "analytic",
		References:        []string{"f1"},
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "" {
		t.Errorf("expected no forced verdict, got %q", result.ForceVerdict)
	}
}

func TestEdgeCase_AllEvidenceRejected(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, _ string) []types.EvidenceRecord {
			return []types.EvidenceRecord{
				{Id: "ev-1", Status: "rejected"},
				{Id: "ev-2", Status: "expired"},
			}
		},
	}
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "empirical claim",
		EpistemicCategory: "empirical",
		References:        nil,
	}

	result := evaluateEdgeCases(ctx, evalCtx, claim)
	if result.ForceVerdict != "abstain" {
		t.Errorf("expected abstain when all evidence rejected, got %q", result.ForceVerdict)
	}
}
