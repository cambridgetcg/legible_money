package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ============================================================================
// Prophetic Detection Tests
// ============================================================================

func TestDetectPropheticFacts_BarelyAcceptedWithCitations(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact that was barely accepted (base confidence near threshold)
	// but has since gained many citations
	fact := createTestFact("fact-prophetic", "physics", "empirical", types.FactStatusVerified, 640_000)
	fact.BaseConfidence = 620_000 // barely above 600k accept threshold
	fact.CitationCount = 15       // above PropheticCitationThreshold (10)
	fact.CreatedAtHeight = 10
	k.SetFact(ctx, fact)

	k.DetectPropheticFacts(ctx, 100_000) // well after creation

	updated, found := k.GetFact(ctx, "fact-prophetic")
	if !found {
		t.Fatal("fact not found")
	}
	if !updated.IsProphetic {
		t.Error("expected fact to be marked as prophetic")
	}
	if updated.Confidence <= fact.Confidence {
		t.Error("prophetic detection should boost confidence")
	}
	if updated.CitationGrowthRate == 0 {
		t.Error("expected non-zero citation growth rate")
	}
}

func TestDetectPropheticFacts_HighConfidenceNotProphetic(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// High base confidence → not a "barely accepted" claim
	fact := createTestFact("fact-strong", "physics", "empirical", types.FactStatusVerified, 800_000)
	fact.BaseConfidence = 800_000
	fact.CitationCount = 50
	k.SetFact(ctx, fact)

	k.DetectPropheticFacts(ctx, 100_000)

	updated, _ := k.GetFact(ctx, "fact-strong")
	if updated.IsProphetic {
		t.Error("high-confidence fact should not be marked prophetic")
	}
}

func TestDetectPropheticFacts_LowCitationsNotProphetic(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Barely accepted but no citations → not vindicated yet
	fact := createTestFact("fact-lonely", "physics", "empirical", types.FactStatusVerified, 610_000)
	fact.BaseConfidence = 610_000
	fact.CitationCount = 3 // below threshold
	k.SetFact(ctx, fact)

	k.DetectPropheticFacts(ctx, 100_000)

	updated, _ := k.GetFact(ctx, "fact-lonely")
	if updated.IsProphetic {
		t.Error("low-citation barely-accepted fact should not be prophetic")
	}
}

func TestDetectPropheticFacts_AlreadyPropheticSkipped(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-already", "physics", "empirical", types.FactStatusVerified, 800_000)
	fact.BaseConfidence = 620_000
	fact.CitationCount = 20
	fact.IsProphetic = true // already marked
	fact.Confidence = 800_000
	k.SetFact(ctx, fact)

	k.DetectPropheticFacts(ctx, 200_000)

	updated, _ := k.GetFact(ctx, "fact-already")
	// Confidence shouldn't change (already prophetic, detection skipped)
	if updated.Confidence != 800_000 {
		t.Errorf("already-prophetic fact confidence should not change, got %d", updated.Confidence)
	}
}

// ============================================================================
// Prophetic Growth Tests
// ============================================================================

func TestApplyPropheticGrowth_DoublesGrowth(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create two facts: one prophetic, one normal, both starting at same confidence
	propheticFact := createTestFact("fact-p", "physics", "empirical", types.FactStatusVerified, 700_000)
	propheticFact.IsProphetic = true
	propheticFact.IsProvisional = false
	k.SetFact(ctx, propheticFact)

	normalFact := createTestFact("fact-n", "physics", "empirical", types.FactStatusVerified, 700_000)
	normalFact.IsProvisional = false
	k.SetFact(ctx, normalFact)

	height := uint64(100_000)

	// Apply prophetic growth
	k.ApplyPropheticGrowth(ctx, height)
	// Apply normal growth
	k.ApplyConfidenceGrowth(ctx, height)

	pFact, _ := k.GetFact(ctx, "fact-p")
	nFact, _ := k.GetFact(ctx, "fact-n")

	// Prophetic should have grown more
	propheticGrowth := pFact.Confidence - 700_000
	normalGrowth := nFact.Confidence - 700_000

	if propheticGrowth <= normalGrowth {
		t.Errorf("prophetic growth (%d) should exceed normal growth (%d)", propheticGrowth, normalGrowth)
	}
}

func TestApplyPropheticGrowth_CappedAtPropheticCap(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-capped", "physics", "empirical", types.FactStatusVerified, 910_000)
	fact.IsProphetic = true
	k.SetFact(ctx, fact)

	k.ApplyPropheticGrowth(ctx, 100_000)

	updated, _ := k.GetFact(ctx, "fact-capped")
	if updated.Confidence > types.PropheticConfidenceCap {
		t.Errorf("prophetic growth should be capped at %d, got %d", types.PropheticConfidenceCap, updated.Confidence)
	}
}

// ============================================================================
// Novelty-Aware Pruning Tests
// ============================================================================

func TestIsFactImmune_PropheticFact(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-immune-prophetic", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.IsProphetic = true
	fact.CitationCount = 5 // below normal immunity threshold
	k.SetFact(ctx, fact)

	if !k.IsFactImmune(ctx, fact) {
		t.Error("prophetic facts should be immune from pruning")
	}
}

func TestGetGracePeriodForFact_NovelExtended(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	normalFact := createTestFact("fact-normal", "physics", "empirical", types.FactStatusVerified, 600_000)
	novelFact := createTestFact("fact-novel", "physics", "empirical", types.FactStatusVerified, 600_000)
	novelFact.NoveltyScore = 600_000 // above NoveltyImmunityThreshold (500k)

	normalGrace := k.GetGracePeriodForFact(ctx, normalFact)
	novelGrace := k.GetGracePeriodForFact(ctx, novelFact)

	if novelGrace <= normalGrace {
		t.Errorf("novel fact grace (%d) should exceed normal grace (%d)", novelGrace, normalGrace)
	}
	if novelGrace != normalGrace*types.NoveltyGraceMultiplier {
		t.Errorf("expected novel grace = normal × %d, got %d (normal=%d)", types.NoveltyGraceMultiplier, novelGrace, normalGrace)
	}
}

func TestGetGracePeriodForFact_PropheticExtended(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-prophetic-grace", "physics", "empirical", types.FactStatusVerified, 600_000)
	fact.IsProphetic = true

	normalFact := createTestFact("fact-baseline", "physics", "empirical", types.FactStatusVerified, 600_000)

	propheticGrace := k.GetGracePeriodForFact(ctx, fact)
	normalGrace := k.GetGracePeriodForFact(ctx, normalFact)

	if propheticGrace <= normalGrace {
		t.Errorf("prophetic fact grace (%d) should exceed normal (%d)", propheticGrace, normalGrace)
	}
}

func TestGetGracePeriodForFact_LowNoveltyNotExtended(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-low-novelty", "physics", "empirical", types.FactStatusVerified, 600_000)
	fact.NoveltyScore = 100_000 // below threshold

	normalFact := createTestFact("fact-no-novelty", "physics", "empirical", types.FactStatusVerified, 600_000)

	novelGrace := k.GetGracePeriodForFact(ctx, fact)
	normalGrace := k.GetGracePeriodForFact(ctx, normalFact)

	if novelGrace != normalGrace {
		t.Errorf("low-novelty fact should not get extended grace: novel=%d normal=%d", novelGrace, normalGrace)
	}
}

// ============================================================================
// KMS Novelty Weight Tests
// ============================================================================

func TestCalculateKMS_NoveltyBoost(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Two facts identical except for novelty score
	normalFact := createTestFact("fact-normal-kms", "physics", "empirical", types.FactStatusVerified, 600_000)
	normalFact.CitationCount = 5
	k.SetFact(ctx, normalFact)

	novelFact := createTestFact("fact-novel-kms", "physics", "empirical", types.FactStatusVerified, 600_000)
	novelFact.CitationCount = 5
	novelFact.NoveltyScore = 800_000 // high novelty
	novelFact.BridgeScore = 50_000   // some bridge value
	k.SetFact(ctx, novelFact)

	normalKMS := k.CalculateKMS(ctx, normalFact)
	novelKMS := k.CalculateKMS(ctx, novelFact)

	if novelKMS <= normalKMS {
		t.Errorf("novel fact KMS (%d) should exceed normal (%d)", novelKMS, normalKMS)
	}
}

func TestCalculateKMS_PropheticBoost(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	normalFact := createTestFact("fact-normal-p", "physics", "empirical", types.FactStatusVerified, 600_000)
	k.SetFact(ctx, normalFact)

	propheticFact := createTestFact("fact-prophetic-p", "physics", "empirical", types.FactStatusVerified, 600_000)
	propheticFact.IsProphetic = true
	k.SetFact(ctx, propheticFact)

	normalKMS := k.CalculateKMS(ctx, normalFact)
	propheticKMS := k.CalculateKMS(ctx, propheticFact)

	if propheticKMS <= normalKMS {
		t.Errorf("prophetic fact KMS (%d) should exceed normal (%d)", propheticKMS, normalKMS)
	}
}

// ============================================================================
// Decay Modulation Tests
// ============================================================================

func TestDecayModulation_NovelSlowerDecay(t *testing.T) {
	// Novel facts should decay slower (2× half-life)
	baseConf := uint64(800_000)
	blocks := uint64(3_000_000) // significant age

	ep := types.DefaultExtendedParams()
	normalDecayed := keeper.CalculateDecayedConfidence(baseConf, blocks, "empirical", ep)
	novelDecayed := keeper.CalculateDecayedConfidenceNovel(baseConf, blocks, "empirical", types.NoveltyDecayHalfLifeMultiplier, ep)

	if novelDecayed <= normalDecayed {
		t.Errorf("novel decay (%d) should be slower (higher confidence) than normal (%d)", novelDecayed, normalDecayed)
	}
}

func TestDecayModulation_MultiplierOne(t *testing.T) {
	// Multiplier of 1 should equal standard decay
	baseConf := uint64(800_000)
	blocks := uint64(3_000_000)

	ep := types.DefaultExtendedParams()
	standard := keeper.CalculateDecayedConfidence(baseConf, blocks, "empirical", ep)
	withMult1 := keeper.CalculateDecayedConfidenceNovel(baseConf, blocks, "empirical", 1, ep)

	if standard != withMult1 {
		t.Errorf("multiplier=1 should equal standard: standard=%d mult1=%d", standard, withMult1)
	}
}

func TestDecayModulation_EternalUnaffected(t *testing.T) {
	// Eternal categories should be unaffected by multiplier
	baseConf := uint64(800_000)
	blocks := uint64(3_000_000)

	ep := types.DefaultExtendedParams()
	standard := keeper.CalculateDecayedConfidence(baseConf, blocks, "analytic", ep)
	novel := keeper.CalculateDecayedConfidenceNovel(baseConf, blocks, "analytic", 5, ep)

	if standard != novel {
		t.Errorf("eternal category should be unaffected by multiplier: standard=%d novel=%d", standard, novel)
	}
}

// ============================================================================
// Extended Params Validation Tests
// ============================================================================

func TestNoveltyParamsValidation(t *testing.T) {
	ep := types.DefaultExtendedParams()

	// Default should be valid
	if err := ep.ValidateExtendedParams(); err != nil {
		t.Errorf("default params should be valid: %v", err)
	}

	// Invalid: novelty bonus > 1M
	invalid := ep
	invalid.NoveltyMaxBonusBps = 2_000_000
	if err := invalid.ValidateExtendedParams(); err == nil {
		t.Error("novelty_max_bonus_bps > 1M should fail validation")
	}

	// Invalid: prophetic cap > 1M
	invalid2 := ep
	invalid2.PropheticConfidenceCapBps = 2_000_000
	if err := invalid2.ValidateExtendedParams(); err == nil {
		t.Error("prophetic_confidence_cap_bps > 1M should fail validation")
	}

	// Invalid: decay multiplier = 0
	invalid3 := ep
	invalid3.NoveltyDecayHalfLifeMultiplier = 0
	if err := invalid3.ValidateExtendedParams(); err == nil {
		t.Error("novelty_decay_half_life_multiplier = 0 should fail validation")
	}
}

func TestNoveltyKMSWeightsSum(t *testing.T) {
	ep := types.DefaultExtendedParams()
	sum := ep.ContributionWeight + ep.UtilizationWeight + ep.TrajectoryWeight + ep.MaturityWeight + ep.NoveltyWeight
	if sum != 1_000_000 {
		t.Errorf("KMS weights (including novelty) should sum to 1,000,000, got %d", sum)
	}
}
