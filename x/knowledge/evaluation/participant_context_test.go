package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func makeEvalCtxWithEvidence(evidences map[string][]types.EvidenceRecord) *EvaluationContext {
	return &EvaluationContext{
		ExtendedParams: types.DefaultExtendedParams(),
		GetEvidenceByClaimId: func(_ sdk.Context, claimId string) []types.EvidenceRecord {
			if evidences != nil {
				return evidences[claimId]
			}
			return nil
		},
	}
}

func TestParticipantContext_DerivableCategory_NoAdjustment(t *testing.T) {
	ctx := mockCtx()

	for _, cat := range []string{"analytic", "formal", "protocol", "computational"} {
		evalCtx := makeEvalCtxWithEvidence(nil)
		claim := types.Claim{
			Id:                "c1",
			ClaimText:         "Derivable claim",
			EpistemicCategory: cat,
		}

		result := evaluateParticipantContext(ctx, evalCtx, claim)

		if result.Adjustment != 0 {
			t.Errorf("[%s] expected 0 adjustment for derivable category, got %d", cat, result.Adjustment)
		}
		if result.Ceiling != 0 {
			t.Errorf("[%s] expected no ceiling for derivable category, got %d", cat, result.Ceiling)
		}
		if result.Name != "participant_context" {
			t.Errorf("[%s] expected stage name participant_context, got %s", cat, result.Name)
		}
	}
}

func TestParticipantContext_ObservableWithHumanEvidence_Bonus(t *testing.T) {
	ctx := mockCtx()
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "experiment", Status: "verified", Reliability: 800_000},
			{Id: "ev-2", EvidenceType: "oracle_feed", Status: "verified", Reliability: 700_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Empirical claim with human evidence",
		EpistemicCategory: "empirical",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	// 2 human evidence items × 10,000 bps = 20,000 bps bonus
	expectedBonus := int64(20_000)
	if result.Adjustment != expectedBonus {
		t.Errorf("expected +%d bonus, got %d", expectedBonus, result.Adjustment)
	}
	if result.Ceiling != 0 {
		t.Errorf("expected no ceiling when human evidence present, got %d", result.Ceiling)
	}
}

func TestParticipantContext_ObservableWithHumanEvidence_CappedBonus(t *testing.T) {
	ctx := mockCtx()
	// 5 human evidence items × 10,000 = 50,000 — should be capped at 30,000
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "experiment", Status: "verified", Reliability: 800_000},
			{Id: "ev-2", EvidenceType: "oracle_feed", Status: "verified", Reliability: 700_000},
			{Id: "ev-3", EvidenceType: "document", Status: "verified", Reliability: 600_000},
			{Id: "ev-4", EvidenceType: "attestation", Status: "verified", Reliability: 500_000},
			{Id: "ev-5", EvidenceType: "experiment", Status: "verified", Reliability: 900_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Well-grounded empirical claim",
		EpistemicCategory: "historical",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	if result.Adjustment != int64(types.MaxHumanEvidenceBonusBpsDefault) {
		t.Errorf("expected capped bonus %d, got %d", types.MaxHumanEvidenceBonusBpsDefault, result.Adjustment)
	}
}

func TestParticipantContext_ObservableWithOnlyAgentEvidence_PenaltyAndCeiling(t *testing.T) {
	ctx := mockCtx()
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 1_000_000},
			{Id: "ev-2", EvidenceType: "computation", Status: "verified", Reliability: 950_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Empirical claim with only agent evidence",
		EpistemicCategory: "empirical",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	expectedPenalty := -int64(types.HumanEvidenceGapPenaltyBpsDefault)
	if result.Adjustment != expectedPenalty {
		t.Errorf("expected %d penalty, got %d", expectedPenalty, result.Adjustment)
	}
	if result.Ceiling != types.AgentOnlyEmpiricalCeilingDefault {
		t.Errorf("expected ceiling %d, got %d", types.AgentOnlyEmpiricalCeilingDefault, result.Ceiling)
	}
}

func TestParticipantContext_ObservableWithMixedEvidence_Bonus(t *testing.T) {
	ctx := mockCtx()
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 1_000_000},
			{Id: "ev-2", EvidenceType: "experiment", Status: "verified", Reliability: 800_000},
			{Id: "ev-3", EvidenceType: "computation", Status: "verified", Reliability: 950_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Empirical claim with mixed evidence",
		EpistemicCategory: "social",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	// 1 human evidence item (experiment) × 10,000 = 10,000 bonus
	// Mixed evidence should get bonus (human evidence present), not penalty
	if result.Adjustment != int64(types.HumanEvidenceBonusBpsDefault) {
		t.Errorf("expected +%d bonus for mixed evidence, got %d", types.HumanEvidenceBonusBpsDefault, result.Adjustment)
	}
	if result.Ceiling != 0 {
		t.Errorf("expected no ceiling for mixed evidence, got %d", result.Ceiling)
	}
}

func TestParticipantContext_ObservableWithNoEvidence_Deferred(t *testing.T) {
	ctx := mockCtx()
	evalCtx := makeEvalCtxWithEvidence(nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Empirical claim without evidence",
		EpistemicCategory: "empirical",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	// No evidence: deferred to evidence stage
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment when no evidence, got %d", result.Adjustment)
	}
	if result.Ceiling != 0 {
		t.Errorf("expected no ceiling when no evidence, got %d", result.Ceiling)
	}
}

func TestParticipantContext_AllEvidenceRejected_Deferred(t *testing.T) {
	ctx := mockCtx()
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "experiment", Status: "rejected", Reliability: 800_000},
			{Id: "ev-2", EvidenceType: "on_chain", Status: "expired", Reliability: 1_000_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Claim with rejected evidence",
		EpistemicCategory: "replicated",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment for all-rejected evidence, got %d", result.Adjustment)
	}
	if result.Ceiling != 0 {
		t.Errorf("expected no ceiling for all-rejected evidence, got %d", result.Ceiling)
	}
}

func TestParticipantContext_NilEvalCtxEvidenceFunc(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{} // GetEvidenceByClaimId is nil

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Observable claim",
		EpistemicCategory: "predictive",
	}

	result := evaluateParticipantContext(ctx, evalCtx, claim)

	// Nil evidence function: treated as no evidence → deferred
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment for nil evidence func, got %d", result.Adjustment)
	}
}

func TestParticipantContext_AllObservableCategories(t *testing.T) {
	ctx := mockCtx()
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 1_000_000},
		},
	}
	evalCtx := makeEvalCtxWithEvidence(evidences)

	// All observable categories should get the penalty when only agent evidence
	for _, cat := range []string{"empirical", "historical", "predictive", "social", "replicated"} {
		claim := types.Claim{
			Id:                "c1",
			ClaimText:         "Observable claim",
			EpistemicCategory: cat,
		}

		result := evaluateParticipantContext(ctx, evalCtx, claim)

		if result.Adjustment >= 0 {
			t.Errorf("[%s] expected negative adjustment for agent-only evidence, got %d", cat, result.Adjustment)
		}
		if result.Ceiling == 0 {
			t.Errorf("[%s] expected ceiling for agent-only evidence, got 0", cat)
		}
	}
}
