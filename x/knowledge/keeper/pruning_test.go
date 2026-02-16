package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ============================================================================
// Helper: create a test fact with defaults
// ============================================================================

func createTestFact(id, domain, category, status string, confidence uint64) types.Fact {
	return types.Fact{
		Id:                   id,
		ClaimText:            "test claim " + id,
		Domain:               domain,
		EpistemicCategory:    category,
		Confidence:           confidence,
		BaseConfidence:       confidence,
		Status:               status,
		Submitter:            testSubmitterAddr,
		CreatedAtHeight:      10,
		VerifiedAtHeight:     10,
		LastVerifiedHeight:   10,
		LastConfidenceUpdate: 10,
	}
}

// ============================================================================
// KMS Calculation Tests
// ============================================================================

func TestCalculateKMS_HighContribution(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact with 25 citations (high contribution)
	fact := createTestFact("fact-high-contrib", "mathematics", "empirical", types.FactStatusVerified, 800_000)
	fact.CitationCount = 25
	fact.LastCitedBlock = uint64(ctx.BlockHeight())
	fact.LastUtilizedBlock = uint64(ctx.BlockHeight())
	k.SetFact(ctx, fact)

	// Add 12 incoming support references to boost fundamentality
	for i := 0; i < 12; i++ {
		refID := "ref-" + string(rune('a'+i))
		refFact := createTestFact(refID, "mathematics", "empirical", types.FactStatusVerified, 700_000)
		k.SetFact(ctx, refFact)
		k.SetFactReference(ctx, types.FactReference{
			FromFactId:      refID,
			ToFactId:        fact.Id,
			ReferenceType:   "supports",
			Weight:          100,
			CreatedAtHeight: 10,
		})
	}

	kms := k.CalculateKMS(ctx, fact)
	if kms < 300_000 {
		t.Errorf("expected high KMS for well-cited fact, got %d", kms)
	}
}

func TestCalculateKMS_ZeroUtilization(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact that has never been cited
	fact := createTestFact("fact-no-util", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	fact.CitationCount = 0
	fact.LastCitedBlock = 0
	fact.LastUtilizedBlock = 0
	k.SetFact(ctx, fact)

	kms := k.CalculateKMS(ctx, fact)
	// Should have low utilization score but some maturity/trajectory
	if kms > 500_000 {
		t.Errorf("expected low KMS for never-cited fact, got %d", kms)
	}
}

func TestCalculateKMS_DecayingTrajectory(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Decaying: confidence dropped from 800k to 300k
	fact := createTestFact("fact-decaying", "mathematics", "empirical", types.FactStatusVerified, 300_000)
	fact.BaseConfidence = 800_000
	fact.CitationCount = 5
	fact.LastCitedBlock = uint64(ctx.BlockHeight())
	k.SetFact(ctx, fact)

	// Stable: confidence == base
	stableFact := createTestFact("fact-stable", "mathematics", "empirical", types.FactStatusVerified, 800_000)
	stableFact.BaseConfidence = 800_000
	stableFact.CitationCount = 5
	stableFact.LastCitedBlock = uint64(ctx.BlockHeight())
	k.SetFact(ctx, stableFact)

	kms := k.CalculateKMS(ctx, fact)
	stableKms := k.CalculateKMS(ctx, stableFact)

	if kms >= stableKms {
		t.Errorf("decaying fact KMS (%d) should be less than stable fact KMS (%d)", kms, stableKms)
	}
}

func TestCalculateKMS_MatureCanonical(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-canonical", "mathematics", "empirical", types.FactStatusVerified, 900_000)
	fact.Maturity = types.MaturityCanonical
	fact.SurvivedChallenges = 3
	fact.CitationCount = 10
	fact.LastCitedBlock = uint64(ctx.BlockHeight())
	fact.CreatedAtHeight = 1 // very old
	k.SetFact(ctx, fact)

	kms := k.CalculateKMS(ctx, fact)
	if kms < 400_000 {
		t.Errorf("expected high KMS for canonical mature fact, got %d", kms)
	}
}

func TestCalculateKMS_WeightSum(t *testing.T) {
	ep := types.DefaultExtendedParams()
	sum := ep.ContributionWeight + ep.UtilizationWeight + ep.TrajectoryWeight + ep.MaturityWeight
	if sum != 1_000_000 {
		t.Errorf("KMS weights should sum to 1,000,000, got %d", sum)
	}
}

func TestCalculateKMS_PatronageBonusApplied(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-patron", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	fact.PatronageBonus = 200_000
	k.SetFact(ctx, fact)

	kmsWithBonus := k.CalculateKMS(ctx, fact)

	factNoBonus := fact
	factNoBonus.PatronageBonus = 0
	kmsWithout := k.CalculateKMS(ctx, factNoBonus)

	if kmsWithBonus <= kmsWithout {
		t.Errorf("patronage bonus should increase KMS: with=%d, without=%d", kmsWithBonus, kmsWithout)
	}
	if kmsWithBonus-kmsWithout != 200_000 {
		t.Errorf("patronage bonus should add exactly 200000: diff=%d", kmsWithBonus-kmsWithout)
	}
}

// ============================================================================
// Immunity Tests
// ============================================================================

func TestIsFactImmune_EternalCategory(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for _, category := range []string{"analytic", "formal", "protocol", "computational"} {
		fact := createTestFact("fact-"+category, "mathematics", category, types.FactStatusVerified, 800_000)
		k.SetFact(ctx, fact)
		if !k.IsFactImmune(ctx, fact) {
			t.Errorf("expected %s category to be immune", category)
		}
	}
}

func TestIsFactImmune_HighCitations(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-cited", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	fact.CitationCount = 50
	k.SetFact(ctx, fact)

	if !k.IsFactImmune(ctx, fact) {
		t.Error("expected fact with 50+ citations to be immune")
	}
}

func TestIsFactImmune_EmpiricalNotImmune(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-emp", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	fact.CitationCount = 5
	k.SetFact(ctx, fact)

	if k.IsFactImmune(ctx, fact) {
		t.Error("expected empirical fact with low citations to NOT be immune")
	}
}

func TestIsFactImmune_TerminalStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for _, status := range []string{types.FactStatusExpired, types.FactStatusDisproven, types.FactStatusPruned, types.FactStatusRevoked} {
		fact := createTestFact("fact-"+status, "mathematics", "empirical", status, 100_000)
		k.SetFact(ctx, fact)
		if !k.IsFactImmune(ctx, fact) {
			t.Errorf("expected terminal status %s to be immune", status)
		}
	}
}

// ============================================================================
// Pruning Lifecycle Tests
// ============================================================================

func TestPruningLifecycle_AtRisk(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.PruningThresholdBps = 100_000 // 10%
	k.SetExtendedParams(ctx, ep)

	// Low-KMS fact: no citations, no utilization
	fact := createTestFact("fact-lowkms", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	fact.CitationCount = 0
	fact.LastCitedBlock = 0
	fact.LastUtilizedBlock = 0
	k.SetFact(ctx, fact)

	processed := k.PruneUnusedFactsInEndBlocker(ctx, 20)
	if processed == 0 {
		t.Fatal("expected at least one fact to be processed")
	}

	updatedFact, found := k.GetFact(ctx, "fact-lowkms")
	if !found {
		t.Fatal("fact should still exist")
	}
	if updatedFact.Status != types.FactStatusAtRisk {
		t.Errorf("expected status at_risk, got %s", updatedFact.Status)
	}
	if updatedFact.PruningGraceStart == 0 {
		t.Error("expected PruningGraceStart to be set")
	}
}

func TestPruningLifecycle_Rescue(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// At_risk fact with high citations (will pass KMS check)
	fact := createTestFact("fact-rescue", "mathematics", "empirical", types.FactStatusAtRisk, 800_000)
	fact.PruningGraceStart = uint64(ctx.BlockHeight())
	fact.CitationCount = 20
	fact.LastCitedBlock = uint64(ctx.BlockHeight())
	fact.LastUtilizedBlock = uint64(ctx.BlockHeight())
	k.SetFact(ctx, fact)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.PruningThresholdBps = 100_000 // 10%
	k.SetExtendedParams(ctx, ep)

	k.PruneUnusedFactsInEndBlocker(ctx, 20)

	updatedFact, found := k.GetFact(ctx, "fact-rescue")
	if !found {
		t.Fatal("fact should still exist")
	}
	if updatedFact.Status != types.FactStatusVerified {
		t.Errorf("expected rescued to verified, got %s", updatedFact.Status)
	}
	if updatedFact.PruningGraceStart != 0 {
		t.Error("expected PruningGraceStart cleared after rescue")
	}
}

func TestPruningLifecycle_Archive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// At_risk fact with grace period expired
	fact := createTestFact("fact-archive", "mathematics", "predictive", types.FactStatusAtRisk, 600_000)
	fact.PruningGraceStart = 1
	fact.CitationCount = 0
	fact.LastCitedBlock = 0
	fact.LastUtilizedBlock = 0
	k.SetFact(ctx, fact)

	// Move to a height past grace period
	ctx = ctx.WithBlockHeight(2_000_000)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.PruningThresholdBps = 500_000 // high threshold, fact can't pass
	k.SetExtendedParams(ctx, ep)

	k.PruneUnusedFactsInEndBlocker(ctx, 20)

	updatedFact, found := k.GetFact(ctx, "fact-archive")
	if !found {
		t.Fatal("fact should still exist (archived, not deleted)")
	}
	if updatedFact.Status != types.FactStatusPruned {
		t.Errorf("expected status pruned, got %s", updatedFact.Status)
	}
}

func TestPruningLifecycle_HardDelete(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Pruned fact with archive retention expired
	fact := createTestFact("fact-delete", "mathematics", "predictive", types.FactStatusPruned, 100_000)
	fact.PruningGraceStart = 1
	k.SetFact(ctx, fact)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.ArchiveRetentionBlocks = 100_000
	k.SetExtendedParams(ctx, ep)

	ctx = ctx.WithBlockHeight(10_000_000)

	k.PruneUnusedFactsInEndBlocker(ctx, 20)

	_, found := k.GetFact(ctx, "fact-delete")
	if found {
		t.Error("expected fact to be hard deleted")
	}
}

func TestPruningLifecycle_CategoryScaledGrace(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Predictive: HalfLifeBlocks=20,000, grace = max(1,028,160, 40,000) = 1,028,160
	gracePredictive := k.GetGracePeriod(ctx, "predictive")

	// Empirical: HalfLifeBlocks=1,000,000, grace = max(1,028,160, 2,000,000) = 2,000,000
	graceEmpirical := k.GetGracePeriod(ctx, "empirical")

	if graceEmpirical <= gracePredictive {
		t.Errorf("empirical grace (%d) should be > predictive grace (%d)", graceEmpirical, gracePredictive)
	}
	if graceEmpirical != 2_000_000 {
		t.Errorf("expected empirical grace = 2,000,000, got %d", graceEmpirical)
	}
}

// ============================================================================
// Patronage Tests
// ============================================================================

func TestPatronizeFact_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create at_risk fact
	fact := createTestFact("fact-atrisk", "mathematics", "empirical", types.FactStatusAtRisk, 600_000)
	fact.PruningGraceStart = uint64(ctx.BlockHeight())
	fact.CitationCount = 0
	k.SetFact(ctx, fact)

	ms := keeper.NewMsgServerImpl(k)
	resp, err := ms.PatronizeFact(sdk.WrapSDKContext(ctx), &types.MsgPatronizeFact{
		Patron: testSubmitterAddr,
		FactId: "fact-atrisk",
		Amount: "10000000", // 10 LGM
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PatronageId == "" {
		t.Error("expected patronage ID in response")
	}
	if resp.NewKmsBonus == 0 {
		t.Error("expected non-zero KMS bonus")
	}

	// Verify patronage record was created
	record, found := k.GetPatronageRecord(ctx, "fact-atrisk", testSubmitterAddr)
	if !found {
		t.Fatal("expected patronage record to exist")
	}
	if record.Amount != "10000000" {
		t.Errorf("expected amount 10000000, got %s", record.Amount)
	}
}

func TestPatronizeFact_NotAtRisk(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Verified fact (NOT at_risk)
	fact := createTestFact("fact-active", "mathematics", "empirical", types.FactStatusVerified, 800_000)
	k.SetFact(ctx, fact)

	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.PatronizeFact(sdk.WrapSDKContext(ctx), &types.MsgPatronizeFact{
		Patron: testSubmitterAddr,
		FactId: "fact-active",
		Amount: "10000000",
	})
	if err == nil {
		t.Error("expected error when patronizing non-at-risk fact")
	}
}

func TestPatronizeFact_SettlementOnPrune(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create patronage record
	record := types.PatronageRecord{
		FactId:    "fact-todie",
		Patron:    testSubmitterAddr,
		Amount:    "5000000",
		CreatedAt: 10,
		ExpiresAt: 1000,
		Settled:   false,
	}
	k.SetPatronageRecord(ctx, record)

	// Create the fact
	fact := createTestFact("fact-todie", "mathematics", "empirical", types.FactStatusPruned, 100_000)
	k.SetFact(ctx, fact)

	// Hard delete triggers settlement
	k.HardDeleteFact(ctx, "fact-todie")

	settled, found := k.GetPatronageRecord(ctx, "fact-todie", testSubmitterAddr)
	if !found {
		t.Fatal("patronage record should still exist after settlement")
	}
	if !settled.Settled {
		t.Error("expected patronage to be settled")
	}
}

// ============================================================================
// EndBlocker Integration Tests
// ============================================================================

func TestPruneEndBlocker_AmortizationCap(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.PruningThresholdBps = 900_000 // Very high
	ep.MaxPruningPerBlock = 3
	k.SetExtendedParams(ctx, ep)

	// Create 10 low-KMS facts
	for i := 0; i < 10; i++ {
		fact := createTestFact(
			"fact-batch-"+string(rune('a'+i)),
			"mathematics", "empirical",
			types.FactStatusVerified, 600_000,
		)
		fact.CitationCount = 0
		fact.LastCitedBlock = 0
		k.SetFact(ctx, fact)
	}

	processed := k.PruneUnusedFactsInEndBlocker(ctx, 3)
	if processed > 3 {
		t.Errorf("expected max 3 processed, got %d", processed)
	}
}

func TestPruneEndBlocker_DisabledByDefault(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ep := types.DefaultExtendedParams()
	k.SetExtendedParams(ctx, ep)

	fact := createTestFact("fact-nopruning", "mathematics", "empirical", types.FactStatusVerified, 600_000)
	k.SetFact(ctx, fact)

	processed := k.PruneUnusedFactsInEndBlocker(ctx, 20)
	if processed != 0 {
		t.Errorf("expected 0 processed when pruning disabled, got %d", processed)
	}

	f, found := k.GetFact(ctx, "fact-nopruning")
	if !found {
		t.Fatal("fact should still exist")
	}
	if f.Status != types.FactStatusVerified {
		t.Errorf("expected verified, got %s", f.Status)
	}
}

func TestPruneEndBlocker_ImmuneFactsSkipped(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ep := types.DefaultExtendedParams()
	ep.PruningEnabled = true
	ep.PruningThresholdBps = 900_000
	k.SetExtendedParams(ctx, ep)

	// Eternal category (immune)
	fact := createTestFact("fact-analytic", "mathematics", "analytic", types.FactStatusVerified, 100_000)
	k.SetFact(ctx, fact)

	k.PruneUnusedFactsInEndBlocker(ctx, 20)

	f, found := k.GetFact(ctx, "fact-analytic")
	if !found {
		t.Fatal("immune fact should still exist")
	}
	if f.Status != types.FactStatusVerified {
		t.Errorf("immune fact should remain verified, got %s", f.Status)
	}
}

// ============================================================================
// Utilization Tracking Test
// ============================================================================

func TestIncrementCitationCount_SetsLastUtilizedBlock(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := createTestFact("fact-cite-util", "mathematics", "empirical", types.FactStatusVerified, 800_000)
	k.SetFact(ctx, fact)

	ctx = ctx.WithBlockHeight(500)
	err := k.IncrementCitationCount(ctx, "fact-cite-util")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, found := k.GetFact(ctx, "fact-cite-util")
	if !found {
		t.Fatal("fact should exist")
	}
	if updated.LastUtilizedBlock != 500 {
		t.Errorf("expected LastUtilizedBlock=500, got %d", updated.LastUtilizedBlock)
	}
	if updated.LastCitedBlock != 500 {
		t.Errorf("expected LastCitedBlock=500, got %d", updated.LastCitedBlock)
	}
	if updated.CitationCount != 1 {
		t.Errorf("expected CitationCount=1, got %d", updated.CitationCount)
	}
}
