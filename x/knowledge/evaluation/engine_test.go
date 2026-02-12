package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// fullEvalCtx creates a complete EvaluationContext with configurable state.
func fullEvalCtx(
	facts map[string]types.Fact,
	evidences map[string][]types.EvidenceRecord,
	contradictions map[string][]types.FactReference,
) *EvaluationContext {
	return &EvaluationContext{
		GetClaim: func(_ sdk.Context, id string) (types.Claim, bool) {
			return types.Claim{}, false
		},
		GetFact: func(_ sdk.Context, id string) (types.Fact, bool) {
			f, ok := facts[id]
			return f, ok
		},
		GetSupportingFacts: func(_ sdk.Context, factId string) []types.FactReference {
			return nil
		},
		GetContradictingFacts: func(_ sdk.Context, factId string) []types.FactReference {
			if contradictions != nil {
				return contradictions[factId]
			}
			return nil
		},
		GetEvidenceByClaimId: func(_ sdk.Context, claimId string) []types.EvidenceRecord {
			if evidences != nil {
				return evidences[claimId]
			}
			return nil
		},
		GetZoneConfidenceCeiling: func(_ sdk.Context, zone string) uint64 {
			// Zone ceilings in 0-1,000,000 scale
			switch zone {
			case "propositional", "presburger":
				return 1_000_000
			case "peano":
				return 850_000
			case "set_theory":
				return 800_000
			case "empirical":
				return 700_000
			default:
				return 700_000
			}
		},
		RequiresIncompletenessAck: func(_ sdk.Context, zone string) bool {
			return zone == "peano" || zone == "set_theory"
		},
		GetConfidenceCeiling: func(_ sdk.Context, domain string) (uint64, uint64, error) {
			switch domain {
			case "mathematics":
				return 1_000_000, 0, nil
			case "physics":
				return 990_000, 0, nil
			case "sociology":
				return 850_000, 0, nil
			default:
				return 1_000_000, 0, nil
			}
		},
	}
}

func TestEvaluateClaim_ProtocolClaim_Accept(t *testing.T) {
	ctx := mockCtx()
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Confidence: 900_000},
	}
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 900_000},
		},
	}
	evalCtx := fullEvalCtx(facts, evidences, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "The genesis block hash is 0xabc...",
		EpistemicCategory: "protocol",
		Domain:            "mathematics",
		LogicZone:         "propositional",
		References:        []string{"f1"},
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != "accept" {
		t.Errorf("expected accept, got %s", result.Verdict)
	}
	if result.Confidence == 0 {
		t.Error("expected non-zero confidence")
	}
	if len(result.Stages) != 8 {
		t.Errorf("expected 8 stages, got %d", len(result.Stages))
	}
}

func TestEvaluateClaim_EmptyClaim_Reject(t *testing.T) {
	ctx := mockCtx()
	evalCtx := fullEvalCtx(nil, nil, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "", // empty
		EpistemicCategory: "protocol",
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != "reject" {
		t.Errorf("expected reject for empty claim, got %s", result.Verdict)
	}
}

func TestEvaluateClaim_GodelZone_Abstain(t *testing.T) {
	ctx := mockCtx()
	evalCtx := fullEvalCtx(nil, nil, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Peano claim without grounding",
		EpistemicCategory: "formal",
		Domain:            "mathematics",
		LogicZone:         "peano",
		References:        nil, // no grounding
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != "abstain" {
		t.Errorf("expected abstain for Godel zone without grounding, got %s", result.Verdict)
	}
}

func TestEvaluateClaim_LowConfidence_Reject(t *testing.T) {
	ctx := mockCtx()
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusDisproven},
		"f2": {Id: "f2", Status: types.FactStatusDisproven},
	}
	evalCtx := fullEvalCtx(facts, nil, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Predictive claim with bad refs",
		EpistemicCategory: "predictive",
		Domain:            "sociology",
		LogicZone:         "empirical",
		References:        []string{"f1", "f2"},
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// predictive is evidence-requiring, no evidence → hard gate forces abstain (Stage 6)
	if result.Verdict != "abstain" {
		t.Errorf("expected abstain (hard evidence gate), got %s (confidence: %d)", result.Verdict, result.Confidence)
	}
}

func TestEvaluateClaim_Determinism(t *testing.T) {
	ctx := mockCtx()
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	evidences := map[string][]types.EvidenceRecord{
		"c1": {{Id: "ev1", EvidenceType: "computation", Status: "verified", Reliability: 700_000}},
	}
	evalCtx := fullEvalCtx(facts, evidences, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Deterministic claim",
		EpistemicCategory: "computational",
		Domain:            "computer_science",
		LogicZone:         "propositional",
		References:        []string{"f1"},
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result1, _ := EvaluateClaim(ctx, evalCtx, round, claim)
	result2, _ := EvaluateClaim(ctx, evalCtx, round, claim)

	if result1.Verdict != result2.Verdict {
		t.Errorf("non-deterministic: verdict %s vs %s", result1.Verdict, result2.Verdict)
	}
	if result1.Confidence != result2.Confidence {
		t.Errorf("non-deterministic: confidence %d vs %d", result1.Confidence, result2.Confidence)
	}
}

func TestEvaluateClaim_NilContext(t *testing.T) {
	ctx := mockCtx()
	claim := types.Claim{Id: "c1", ClaimText: "test"}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	_, err := EvaluateClaim(ctx, nil, round, claim)
	if err == nil {
		t.Error("expected error for nil evaluation context")
	}
}

func TestEvaluateClaim_SelfReferential_Abstain(t *testing.T) {
	ctx := mockCtx()
	evalCtx := fullEvalCtx(nil, nil, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "I am my own reference",
		ContentHash:       "hash-xyz",
		EpistemicCategory: "analytic",
		Domain:            "logic",
		References:        []string{"hash-xyz"}, // self-reference
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != "abstain" {
		t.Errorf("expected abstain for self-referential claim, got %s", result.Verdict)
	}
}

func TestEvaluateClaim_AxiomContradiction_Reject(t *testing.T) {
	ctx := mockCtx()
	facts := map[string]types.Fact{
		"axiom-1": {
			Id:             "axiom-1",
			Status:         types.FactStatusDisproven,
			Confidence:     0,
			BaseConfidence: 950_000, // was axiomatic
		},
	}
	evalCtx := fullEvalCtx(facts, nil, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Claim based on disproven axiom",
		EpistemicCategory: "formal",
		Domain:            "mathematics",
		LogicZone:         "propositional",
		References:        []string{"axiom-1"},
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != "reject" {
		t.Errorf("expected reject for axiom contradiction, got %s", result.Verdict)
	}
}

func TestEvaluateClaim_EmpiricalWithEvidence_Accept(t *testing.T) {
	ctx := mockCtx()
	facts := map[string]types.Fact{
		"f1": {Id: "f1", Status: types.FactStatusVerified, Confidence: 800_000},
		"f2": {Id: "f2", Status: types.FactStatusActive, Confidence: 750_000},
	}
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "experiment", Status: "verified", Reliability: 800_000},
			{Id: "ev-2", EvidenceType: "computation", Status: "verified", Reliability: 900_000},
		},
	}
	evalCtx := fullEvalCtx(facts, evidences, nil)

	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Well-supported empirical claim",
		EpistemicCategory: "empirical",
		Domain:            "physics",
		LogicZone:         "empirical",
		References:        []string{"f1", "f2"},
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// empirical base = 500,000
	// evidence bonus (capped at 150,000)
	// 2 verified refs = +60,000
	// ceiling = min(700,000 zone, 990,000 stratum, 990,000 category) = 700,000
	// Total should be capped at 700,000 which is >= 600,000 accept threshold
	if result.Verdict != "accept" {
		t.Errorf("expected accept for well-evidenced empirical claim, got %s (confidence: %d)", result.Verdict, result.Confidence)
	}
	if result.Confidence > 700_000 {
		t.Errorf("confidence should be capped at 700000 (empirical zone), got %d", result.Confidence)
	}
}

func TestEvaluateClaim_InconclusiveRange_Abstain(t *testing.T) {
	ctx := mockCtx()
	evalCtx := fullEvalCtx(nil, nil, nil)

	// social base = 400,000, which is between RejectThreshold (300,000) and AcceptThreshold (600,000)
	claim := types.Claim{
		Id:                "c1",
		ClaimText:         "Inconclusive social claim",
		EpistemicCategory: "social",
		Domain:            "sociology",
		LogicZone:         "empirical",
		References:        []string{"some-ref"}, // has ref so no edge case penalty
	}
	round := types.VerificationRound{Id: "r1", ClaimId: "c1"}

	result, err := EvaluateClaim(ctx, evalCtx, round, claim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// base 400,000, broken ref -10,000, no evidence (has ref though so no penalty)
	// = 390,000, in [300,000, 600,000) → abstain
	if result.Verdict != "abstain" {
		t.Errorf("expected abstain for inconclusive range, got %s (confidence: %d)", result.Verdict, result.Confidence)
	}
}

func TestApplyAdjustment_Overflow(t *testing.T) {
	result := applyAdjustment(999_000, 100_000)
	if result != 1_000_000 {
		t.Errorf("expected 1000000 cap, got %d", result)
	}
}

func TestApplyAdjustment_Underflow(t *testing.T) {
	result := applyAdjustment(10, -100)
	if result != 0 {
		t.Errorf("expected 0 floor, got %d", result)
	}
}

func TestApplyAdjustment_Normal(t *testing.T) {
	result := applyAdjustment(500_000, -100_000)
	if result != 400_000 {
		t.Errorf("expected 400000, got %d", result)
	}
}
