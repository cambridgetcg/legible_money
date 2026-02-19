package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func noveltyEvalCtx(
	facts map[string]*types.Fact,
	domainFactCount map[string]uint64,
	totalDomains uint64,
) *EvaluationContext {
	ctx := fullEvalCtx(facts, nil, nil)
	ctx.GetFactCountByDomain = func(_ sdk.Context, domain string) uint64 {
		if domainFactCount != nil {
			return domainFactCount[domain]
		}
		return 0
	}
	ctx.GetDomainCount = func(_ sdk.Context) uint64 {
		return totalDomains
	}
	return ctx
}

// ==========================================================================
// Information Gain Tests
// ==========================================================================

func TestInformationGain_SparseDomain(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	domainCounts := map[string]uint64{
		"quantum_topology": 2, // very sparse
	}

	claim := &types.Claim{
		Domain:     "quantum_topology",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	gain := calculateInformationGain(ctx, evalCtx, claim)
	if gain == 0 {
		t.Error("sparse domain should produce non-zero information gain")
	}
	// With only 2 facts in 100-cap domain, sparsity is 98% → high info gain
	if gain < 200_000 {
		t.Errorf("expected high info gain for sparse domain, got %d", gain)
	}
}

func TestInformationGain_DenseDomain(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	domainCounts := map[string]uint64{
		"physics": 100, // fully saturated
	}

	claim := &types.Claim{
		Domain:     "physics",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	gain := calculateInformationGain(ctx, evalCtx, claim)
	// Dense domain → low sparsity (10% floor), but geometric mean with ref novelty
	// could push it up. Key: it should be lower than sparse domain.
	sparseFacts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	sparseCtx := noveltyEvalCtx(sparseFacts, map[string]uint64{"rare_field": 2}, 10)
	sparseClaim := &types.Claim{Domain: "rare_field", References: []string{"f1"}}
	sparseGain := calculateInformationGain(ctx, sparseCtx, sparseClaim)
	if gain >= sparseGain {
		t.Errorf("dense domain gain (%d) should be less than sparse (%d)", gain, sparseGain)
	}
}

func TestInformationGain_CrossDomainRefs(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 800_000},
		"f2": {Id: "f2", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000},
		"f3": {Id: "f3", Domain: "biology", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	domainCounts := map[string]uint64{
		"chemistry": 10,
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1", "f2", "f3"}, // 3 unique domains from 3 refs
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	gain := calculateInformationGain(ctx, evalCtx, claim)
	// High reference novelty (3 unique domains / 3 refs = 1.0) + decent sparsity
	if gain < 200_000 {
		t.Errorf("expected high info gain for cross-domain synthesis, got %d", gain)
	}
}

func TestInformationGain_NoRefs(t *testing.T) {
	domainCounts := map[string]uint64{
		"new_domain": 0,
	}

	claim := &types.Claim{
		Domain:     "new_domain",
		References: []string{},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(nil, domainCounts, 10)

	gain := calculateInformationGain(ctx, evalCtx, claim)
	// No refs → halved sparsity bonus
	if gain == 0 {
		t.Error("empty domain should still produce some info gain")
	}
}

// ==========================================================================
// Bridge Score Tests
// ==========================================================================

func TestBridgeScore_VerifiedCrossDomain(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 800_000},
		"f2": {Id: "f2", Domain: "physics", Status: types.FactStatusVerified, Confidence: 900_000},
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1", "f2"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, nil, 0)

	bridge := calculateBridgeScore(ctx, evalCtx, claim)
	if bridge == 0 {
		t.Error("verified cross-domain refs should produce bridge bonus")
	}
	// Two verified cross-domain refs, each above VerifiedBridgeThreshold
	if bridge < types.CrossDomainBridgeBonusBps {
		t.Errorf("expected bridge >= single bridge bonus (%d), got %d", types.CrossDomainBridgeBonusBps, bridge)
	}
}

func TestBridgeScore_UnverifiedCrossDomain(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 400_000}, // below threshold
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, nil, 0)

	bridge := calculateBridgeScore(ctx, evalCtx, claim)
	if bridge != 0 {
		t.Errorf("unverified cross-domain refs should not produce bridge bonus, got %d", bridge)
	}
}

func TestBridgeScore_SameDomain(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 900_000},
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, nil, 0)

	bridge := calculateBridgeScore(ctx, evalCtx, claim)
	if bridge != 0 {
		t.Errorf("same-domain refs should not produce bridge bonus, got %d", bridge)
	}
}

func TestBridgeScore_CappedAtMax(t *testing.T) {
	// Create many verified cross-domain refs
	facts := map[string]*types.Fact{}
	refs := []string{}
	domains := []string{"biology", "physics", "chemistry", "geology", "astronomy", "ecology", "paleontology", "meteorology", "oceanography", "seismology"}
	for _, d := range domains {
		id := "fact-" + d
		facts[id] = &types.Fact{
			Id: id, Domain: d, Status: types.FactStatusVerified, Confidence: 900_000,
		}
		refs = append(refs, id)
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: refs,
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, nil, 0)

	bridge := calculateBridgeScore(ctx, evalCtx, claim)
	if bridge > types.MaxBridgeBonusBps {
		t.Errorf("bridge score should be capped at %d, got %d", types.MaxBridgeBonusBps, bridge)
	}
}

// ==========================================================================
// Novelty Stage Integration Tests
// ==========================================================================

func TestNoveltyStage_EvidenceGate(t *testing.T) {
	claim := &types.Claim{
		Domain: "mathematics",
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(nil, nil, 0)

	// Zero/negative evidence adjustment → novelty gate blocks bonus
	result := evaluateNovelty(ctx, evalCtx, claim, 0) // no evidence → gate blocks
	if result.Adjustment != 0 {
		t.Errorf("low evidence should block novelty bonus, got adjustment %d", result.Adjustment)
	}
}

func TestNoveltyStage_WithGoodEvidence(t *testing.T) {
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000},
	}
	domainCounts := map[string]uint64{
		"chemistry": 5,
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	// Good evidence → novelty bonus should apply
	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	if result.Adjustment <= 0 {
		t.Errorf("good evidence + cross-domain should produce novelty bonus, got %d", result.Adjustment)
	}
}

func TestNoveltyStage_CappedAtMax(t *testing.T) {
	// Create scenario that maximizes both info gain and bridge
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 950_000},
		"f2": {Id: "f2", Domain: "biology", Status: types.FactStatusVerified, Confidence: 950_000},
		"f3": {Id: "f3", Domain: "chemistry", Status: types.FactStatusVerified, Confidence: 950_000},
	}
	domainCounts := map[string]uint64{
		"new_interdisciplinary_field": 0,
	}

	claim := &types.Claim{
		Domain:     "new_interdisciplinary_field",
		References: []string{"f1", "f2", "f3"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 800_000)
	if uint64(result.Adjustment) > types.NoveltyMaxBonusBps {
		t.Errorf("novelty bonus should be capped at %d, got %d", types.NoveltyMaxBonusBps, result.Adjustment)
	}
}

func TestNoveltyStage_NoRefsNoDomain(t *testing.T) {
	claim := &types.Claim{}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(nil, nil, 0)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	// No domain, no refs → some default sparsity but no bridge
	if result.Adjustment < 0 {
		t.Errorf("novelty should never penalize, got %d", result.Adjustment)
	}
}

// ==========================================================================
// FARM-10: Semantic Novelty Floor (Axiom Restating Prevention)
// ==========================================================================

func TestFARM10_LowInfoGainCanonicalRefsPenalized(t *testing.T) {
	// Claim with low info gain + all canonical refs → penalized
	facts := map[string]*types.Fact{
		"MATH-001": {Id: "MATH-001", Domain: "mathematics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
		"MATH-002": {Id: "MATH-002", Domain: "mathematics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
	}
	domainCounts := map[string]uint64{
		"mathematics": 100, // dense domain → low sparsity → low info gain
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: []string{"MATH-001", "MATH-002"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000) // good evidence
	if result.Adjustment >= 0 {
		t.Errorf("FARM-10: axiom-restating claim should be penalized, got adjustment %d", result.Adjustment)
	}
	if result.Adjustment != -int64(evalCtx.ExtendedParams.RestatingPenaltyBps) {
		t.Errorf("FARM-10: expected penalty %d, got %d", -int64(evalCtx.ExtendedParams.RestatingPenaltyBps), result.Adjustment)
	}
}

func TestFARM10_LowInfoGainMixedMaturityNotPenalized(t *testing.T) {
	// Claim with low info gain + mixed maturity refs → NOT penalized
	facts := map[string]*types.Fact{
		"MATH-001": {Id: "MATH-001", Domain: "mathematics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
		"new-001":  {Id: "new-001", Domain: "mathematics", Maturity: types.MaturityEmerging, Confidence: 600_000},
	}
	domainCounts := map[string]uint64{
		"mathematics": 100, // dense → low info gain
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: []string{"MATH-001", "new-001"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	// Mixed maturity → FARM-10 does not trigger; normal novelty score
	if result.Adjustment < 0 {
		t.Errorf("FARM-10: mixed maturity refs should not be penalized, got %d", result.Adjustment)
	}
}

func TestFARM10_HighInfoGainCanonicalRefsNotPenalized(t *testing.T) {
	// Claim with high info gain + all canonical refs → NOT penalized
	facts := map[string]*types.Fact{
		"MATH-001": {Id: "MATH-001", Domain: "mathematics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
	}
	domainCounts := map[string]uint64{
		"new_field": 0, // empty domain → max sparsity → high info gain
	}

	claim := &types.Claim{
		Domain:     "new_field",
		References: []string{"MATH-001"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	if result.Adjustment < 0 {
		t.Errorf("FARM-10: high info gain should not trigger penalty, got %d", result.Adjustment)
	}
}

func TestFARM10_CrossDomainCanonicalRefsNotPenalized(t *testing.T) {
	// Cross-domain claim with canonical refs → NOT penalized (high info gain from synthesis)
	facts := map[string]*types.Fact{
		"MATH-001": {Id: "MATH-001", Domain: "mathematics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
		"PHYS-001": {Id: "PHYS-001", Domain: "physics", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
		"BIO-001":  {Id: "BIO-001", Domain: "biology", Maturity: types.MaturityCanonical, Confidence: 1_000_000},
	}
	domainCounts := map[string]uint64{
		"biophysics": 5, // sparse-ish
	}

	claim := &types.Claim{
		Domain:     "biophysics",
		References: []string{"MATH-001", "PHYS-001", "BIO-001"}, // 3 unique domains
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	// Cross-domain synthesis produces high reference novelty → high info gain
	if result.Adjustment < 0 {
		t.Errorf("FARM-10: cross-domain synthesis should not be penalized, got %d", result.Adjustment)
	}
}

func TestFARM10_NoRefsNotPenalized(t *testing.T) {
	// Claim with no refs → FARM-10 does not trigger (requires len(refs) > 0)
	domainCounts := map[string]uint64{
		"mathematics": 100,
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: []string{},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(nil, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	if result.Adjustment < 0 {
		t.Errorf("FARM-10: no refs should not trigger penalty, got %d", result.Adjustment)
	}
}

func TestFARM10_EstablishedRefsTrigger(t *testing.T) {
	// Established maturity refs (not just canonical) should also trigger FARM-10.
	// Uses 2 same-domain refs → info gain ~224k < 250k threshold.
	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "mathematics", Maturity: types.MaturityEstablished, Confidence: 900_000},
		"f2": {Id: "f2", Domain: "mathematics", Maturity: types.MaturityEstablished, Confidence: 850_000},
	}
	domainCounts := map[string]uint64{
		"mathematics": 100,
	}

	claim := &types.Claim{
		Domain:     "mathematics",
		References: []string{"f1", "f2"},
	}

	ctx := mockCtx()
	evalCtx := noveltyEvalCtx(facts, domainCounts, 10)

	result := evaluateNovelty(ctx, evalCtx, claim, 600_000)
	if result.Adjustment >= 0 {
		t.Errorf("FARM-10: established-only refs in dense domain should be penalized, got %d", result.Adjustment)
	}
}

// ==========================================================================
// AH-3 Verified Bridge Exemption Tests
// ==========================================================================

func TestFactGraph_VerifiedBridgeSkipsPenalty(t *testing.T) {
	facts := map[string]*types.Fact{
		// Cross-domain but well-verified → should skip AH-3 penalty
		"f1": {Id: "f1", Domain: "biology", Status: types.FactStatusVerified, Confidence: 800_000},
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := fullEvalCtx(facts, nil, nil)

	result := evaluateFactGraph(ctx, evalCtx, claim)
	// Verified cross-domain ref: gets support bonus, no AH-3 penalty
	if result.Adjustment < 0 {
		t.Errorf("verified cross-domain bridge should not be penalized, got %d", result.Adjustment)
	}
}

func TestFactGraph_UnverifiedCrossDomainStillPenalized(t *testing.T) {
	facts := map[string]*types.Fact{
		// Cross-domain and low-confidence → still gets AH-3 penalty
		"f1": {Id: "f1", Domain: "biology", Status: types.FactStatusVerified, Confidence: 400_000},
	}

	claim := &types.Claim{
		Domain:     "chemistry",
		References: []string{"f1"},
	}

	ctx := mockCtx()
	evalCtx := fullEvalCtx(facts, nil, nil)

	resultWithUnverified := evaluateFactGraph(ctx, evalCtx, claim)
	// Low-confidence cross-domain ref: gets support bonus but also AH-3 penalty
	// Net might be positive (support > penalty) but penalty was applied
	// Compare with same-domain to verify penalty was applied
	sameDomainFacts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "chemistry", Status: types.FactStatusVerified, Confidence: 400_000},
	}
	sameDomainCtx := fullEvalCtx(sameDomainFacts, nil, nil)
	resultSameDomain := evaluateFactGraph(ctx, sameDomainCtx, claim)

	if resultWithUnverified.Adjustment >= resultSameDomain.Adjustment {
		t.Errorf("unverified cross-domain should score lower than same-domain: cross=%d same=%d",
			resultWithUnverified.Adjustment, resultSameDomain.Adjustment)
	}
}

// ==========================================================================
// Full Pipeline Integration Test
// ==========================================================================

func TestNoveltyStage_InPipeline(t *testing.T) {
	ctx := mockCtx()

	facts := map[string]*types.Fact{
		"f1": {Id: "f1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 900_000},
	}
	evidences := map[string][]types.EvidenceRecord{
		"c1": {
			{Id: "ev-1", EvidenceType: "on_chain", Status: "verified", Reliability: 900_000},
			{Id: "ev-2", EvidenceType: "experiment", Status: "verified", Reliability: 800_000},
		},
	}

	evalCtx := noveltyEvalCtx(facts, map[string]uint64{"chemistry": 3}, 10)
	evalCtx.GetEvidenceByClaimId = func(_ sdk.Context, claimId string) []types.EvidenceRecord {
		return evidences[claimId]
	}

	claim := &types.Claim{
		Id:                "c1",
		ClaimText:         "chemistry bridges physics through quantum mechanics",
		Domain:            "chemistry",
		EpistemicCategory: "empirical",
		References:        []string{"f1"},
		LogicZone:         "empirical",
	}

	result, err := EvaluateClaim(ctx, evalCtx, &types.VerificationRound{}, claim)
	if err != nil {
		t.Fatalf("EvaluateClaim failed: %v", err)
	}

	if len(result.Stages) != 10 {
		t.Errorf("expected 10 stages, got %d", len(result.Stages))
	}

	// Find novelty stage (should be stage index 5, name "novelty")
	var noveltyStage *StageResult
	for i, s := range result.Stages {
		if s.Name == "novelty" {
			noveltyStage = &result.Stages[i]
			break
		}
	}
	if noveltyStage == nil {
		t.Fatal("novelty stage not found in pipeline")
	}

	// With good evidence and cross-domain verified ref, novelty should be positive
	if noveltyStage.Adjustment <= 0 {
		t.Errorf("expected positive novelty adjustment, got %d", noveltyStage.Adjustment)
	}
}
