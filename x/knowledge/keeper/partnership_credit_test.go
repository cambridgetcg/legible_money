package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ============================================================================
// B12-4: Partnership Attribution Tests
// ============================================================================

func TestPartnershipAttribution_ClaimRecordsPartnershipId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_claim_test____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_claim_test____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-claim-test",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create a claim as the agent (who is in a partnership)
	claim := types.Claim{
		Id:                "claim-partner-1",
		ClaimText:         "Water boils at 100C at sea level",
		ContentHash:       "abc123",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         agentAddr,
		StakeAmount:       "1000000",
		Status:            types.ClaimStatusPendingEvaluation,
		SubmittedAtHeight: 100,
	}

	// The partnership lookup happens in SubmitClaim handler, but we can test
	// the attribution directly by checking the keeper flow.
	// Simulate what SubmitClaim does: look up partnership for submitter.
	if partnership, found := mpk.GetActivePartnershipForAddress(ctx, agentAddr); found {
		claim.PartnershipId = partnership.Id
	}
	k.SetClaim(ctx, &claim)

	// Verify claim has partnership_id
	stored, found := k.GetClaim(ctx, "claim-partner-1")
	if !found {
		t.Fatal("claim not found")
	}
	if stored.PartnershipId != "p-claim-test" {
		t.Errorf("expected partnership_id 'p-claim-test', got %q", stored.PartnershipId)
	}
}

func TestPartnershipAttribution_FactInheritsPartnershipId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact with partnership attribution
	fact := createTestFact("fact-partner-1", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.PartnershipId = "p-fact-test"
	k.SetFact(ctx, &fact)

	// Verify fact has partnership_id after retrieval
	stored, found := k.GetFact(ctx, "fact-partner-1")
	if !found {
		t.Fatal("fact not found")
	}
	if stored.PartnershipId != "p-fact-test" {
		t.Errorf("expected partnership_id 'p-fact-test', got %q", stored.PartnershipId)
	}
}

func TestPartnershipAttribution_NoPartnershipLeavesEmpty(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// No partnerships keeper wired
	claim := types.Claim{
		Id:                "claim-no-partner",
		ClaimText:         "Test claim without partnership",
		ContentHash:       "def456",
		Domain:            "mathematics",
		EpistemicCategory: "analytic",
		Submitter:         testSubmitterAddr,
		Status:            types.ClaimStatusPendingEvaluation,
		SubmittedAtHeight: 100,
	}
	k.SetClaim(ctx, &claim)

	stored, found := k.GetClaim(ctx, "claim-no-partner")
	if !found {
		t.Fatal("claim not found")
	}
	if stored.PartnershipId != "" {
		t.Errorf("expected empty partnership_id, got %q", stored.PartnershipId)
	}
}

// ============================================================================
// B12-4: Partnership Citation Stats Tests
// ============================================================================

func TestPartnershipCitationStats_FactCreation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create fact with partnership
	fact := createTestFact("fact-stats-1", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.PartnershipId = "p-stats-1"
	k.SetFact(ctx, &fact)

	// Record fact creation
	k.RecordPartnershipFactCreation(ctx, &fact)

	stats := k.GetPartnershipCitationStats(ctx, "p-stats-1")
	if stats.TotalFactsCreated != 1 {
		t.Errorf("expected 1 fact created, got %d", stats.TotalFactsCreated)
	}
	if len(stats.DomainsCovered) != 1 || stats.DomainsCovered[0] != "physics" {
		t.Errorf("expected domains [physics], got %v", stats.DomainsCovered)
	}

	// Create a second fact in the same domain
	fact2 := createTestFact("fact-stats-2", "physics", "empirical", types.FactStatusVerified, 650_000)
	fact2.PartnershipId = "p-stats-1"
	k.SetFact(ctx, &fact2)
	k.RecordPartnershipFactCreation(ctx, &fact2)

	stats = k.GetPartnershipCitationStats(ctx, "p-stats-1")
	if stats.TotalFactsCreated != 2 {
		t.Errorf("expected 2 facts created, got %d", stats.TotalFactsCreated)
	}
	// Domain should not be duplicated
	if len(stats.DomainsCovered) != 1 {
		t.Errorf("expected 1 domain (deduplicated), got %d", len(stats.DomainsCovered))
	}

	// Create a fact in a different domain
	fact3 := createTestFact("fact-stats-3", "mathematics", "analytic", types.FactStatusVerified, 800_000)
	fact3.PartnershipId = "p-stats-1"
	k.SetFact(ctx, &fact3)
	k.RecordPartnershipFactCreation(ctx, &fact3)

	stats = k.GetPartnershipCitationStats(ctx, "p-stats-1")
	if stats.TotalFactsCreated != 3 {
		t.Errorf("expected 3 facts created, got %d", stats.TotalFactsCreated)
	}
	if len(stats.DomainsCovered) != 2 {
		t.Errorf("expected 2 domains, got %d: %v", len(stats.DomainsCovered), stats.DomainsCovered)
	}
}

func TestPartnershipCitationStats_CitationTracking(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create fact with partnership
	fact := createTestFact("fact-cited-p", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.PartnershipId = "p-cited-1"
	k.SetFact(ctx, &fact)
	k.RecordPartnershipFactCreation(ctx, &fact)

	// Record citations via IncrementCitationCount (which calls RecordPartnershipCitation)
	_ = k.IncrementCitationCount(ctx, "fact-cited-p")
	_ = k.IncrementCitationCount(ctx, "fact-cited-p")
	_ = k.IncrementCitationCount(ctx, "fact-cited-p")

	stats := k.GetPartnershipCitationStats(ctx, "p-cited-1")
	if stats.TotalCitations != 3 {
		t.Errorf("expected 3 citations, got %d", stats.TotalCitations)
	}
	if stats.TotalFactsCreated != 1 {
		t.Errorf("expected 1 fact created, got %d", stats.TotalFactsCreated)
	}
}

func TestPartnershipCitationStats_NoPartnershipSkipped(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create fact WITHOUT partnership
	fact := createTestFact("fact-no-partner-cite", "physics", "empirical", types.FactStatusVerified, 700_000)
	k.SetFact(ctx, &fact)

	// Record creation — should be a no-op
	k.RecordPartnershipFactCreation(ctx, &fact)

	// Increment citation — should not crash or record partnership stats
	_ = k.IncrementCitationCount(ctx, "fact-no-partner-cite")

	// No stats should exist
	stats := k.GetPartnershipCitationStats(ctx, "")
	if stats.TotalFactsCreated != 0 {
		t.Errorf("expected 0 facts for empty partnership, got %d", stats.TotalFactsCreated)
	}
}

// ============================================================================
// B12-4: Partnership Reputation Split Tests
// ============================================================================

func TestPartnershipReputationSplit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_rep_split_____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_rep_split_____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-rep-split",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create a fact submitted by the agent
	fact := createTestFact("fact-rep-1", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.Submitter = agentAddr
	fact.PartnershipId = "p-rep-split"
	k.SetFact(ctx, &fact)

	// Credit partnership reputation
	k.CreditPartnershipReputation(ctx, &fact, true)

	// Both partners should have calibration records
	// The submitter (agent) gets credit through normal UpdateSubmitterCalibration
	// The partner (human) gets credit through CreditPartnershipReputation
	humanCal, found := k.GetCalibrationRecord(ctx, humanAddr)
	if !found || humanCal.TotalClaims == 0 {
		t.Error("expected human partner to have calibration record from partnership credit")
	}
}

func TestPartnershipReputationSplit_NoPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Fact without partnership — CreditPartnershipReputation should be no-op
	fact := createTestFact("fact-no-rep", "physics", "empirical", types.FactStatusVerified, 700_000)
	k.SetFact(ctx, &fact)

	// Should not panic or crash
	k.CreditPartnershipReputation(ctx, &fact, true)
}

func TestPartnershipReputationSplit_HumanSubmitter(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_submit_rep____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_submit_rep____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-human-submit",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create a fact submitted by the HUMAN
	fact := createTestFact("fact-human-rep", "physics", "empirical", types.FactStatusVerified, 700_000)
	fact.Submitter = humanAddr
	fact.PartnershipId = "p-human-submit"
	k.SetFact(ctx, &fact)

	// Credit partnership reputation — agent should get the credit
	k.CreditPartnershipReputation(ctx, &fact, true)

	agentCal, found := k.GetCalibrationRecord(ctx, agentAddr)
	if !found || agentCal.TotalClaims == 0 {
		t.Error("expected agent partner to have calibration record from partnership credit")
	}
}

// ============================================================================
// B12-4: Partnership Profile Query Tests
// ============================================================================

func TestPartnershipProfile(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Build up partnership stats
	for i := 0; i < 5; i++ {
		fact := createTestFact(
			"fact-profile-"+string(rune('a'+i)),
			[]string{"physics", "mathematics", "chemistry"}[i%3],
			"empirical",
			types.FactStatusVerified,
			700_000,
		)
		fact.PartnershipId = "p-profile"
		k.SetFact(ctx, &fact)
		k.RecordPartnershipFactCreation(ctx, &fact)
	}

	// Add citations to some facts
	for i := 0; i < 3; i++ {
		_ = k.IncrementCitationCount(ctx, "fact-profile-a")
	}
	_ = k.IncrementCitationCount(ctx, "fact-profile-b")

	profile := k.GetPartnershipProfile(ctx, "p-profile")
	if profile.PartnershipId != "p-profile" {
		t.Errorf("expected partnership_id 'p-profile', got %q", profile.PartnershipId)
	}
	if profile.TotalFactsCreated != 5 {
		t.Errorf("expected 5 facts, got %d", profile.TotalFactsCreated)
	}
	if profile.TotalCitations != 4 {
		t.Errorf("expected 4 citations, got %d", profile.TotalCitations)
	}
	if len(profile.DomainsCovered) != 3 {
		t.Errorf("expected 3 domains, got %d: %v", len(profile.DomainsCovered), profile.DomainsCovered)
	}
}

// ============================================================================
// B12-4: Fact Creation Path Integration Tests
// ============================================================================

func TestPartnershipAttribution_CompleteRound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_round_test____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_round_test____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-round-test",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create claim with partnership attribution
	claim := types.Claim{
		Id:                "claim-round-p",
		ClaimText:         "Test claim for round completion",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         agentAddr,
		StakeAmount:       "1000000",
		Status:            types.ClaimStatusPending,
		SubmittedAtHeight: 100,
		PartnershipId:     "p-round-test",
	}
	k.SetClaim(ctx, &claim)

	// Create and complete a verification round
	round := types.VerificationRound{
		Id:          "round-p-1",
		ClaimId:     "claim-round-p",
		Phase:       "aggregation",
		StartHeight: 100,
		Commitments: make(map[string]*types.VerificationCommitment),
		Reveals:     make(map[string]*types.VerificationReveal),
	}
	k.SetVerificationRound(ctx, &round)

	result := &types.VerificationResult{
		Verdict:          "accepted",
		Confidence:       700_000,
		ParticipantCount: 3,
	}

	err := k.CompleteRound(ctx, &round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Find the created fact
	factID := keeper.GenerateFactID("claim-round-p", "round-p-1")
	fact, found := k.GetFact(ctx, factID)
	if !found {
		t.Fatal("fact not found after CompleteRound")
	}

	// Verify partnership_id propagated
	if fact.PartnershipId != "p-round-test" {
		t.Errorf("expected partnership_id 'p-round-test', got %q", fact.PartnershipId)
	}

	// Verify partnership stats were recorded
	stats := k.GetPartnershipCitationStats(ctx, "p-round-test")
	if stats.TotalFactsCreated != 1 {
		t.Errorf("expected 1 fact created in partnership stats, got %d", stats.TotalFactsCreated)
	}
}

func TestPartnershipAttribution_ProvisionalToVerified(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_prov_test_____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_prov_test_____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-prov-test",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create a provisional fact with partnership
	fact := types.Fact{
		Id:                "fact-prov-partner",
		ClaimText:         "Provisional test fact",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Confidence:        600_000,
		Status:            types.FactStatusProvisional,
		Submitter:         agentAddr,
		IsProvisional:     true,
		ChallengeWindowEnd: 200,
		PartnershipId:     "p-prov-test",
		CreatedAtHeight:   100,
	}
	k.SetFact(ctx, &fact)

	// Create the claim
	claim := types.Claim{
		Id:                "claim-prov-partner",
		ClaimText:         "Provisional test fact",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         agentAddr,
		StakeAmount:       "1000000",
		Status:            types.ClaimStatusProvisional,
		ProvisionalFactId: "fact-prov-partner",
		ChallengeWindowEnd: 200,
		PartnershipId:     "p-prov-test",
		SubmittedAtHeight: 100,
	}
	k.SetClaim(ctx, &claim)

	// Transition to verified
	err := k.TransitionToVerified(ctx, &claim, 300)
	if err != nil {
		t.Fatalf("TransitionToVerified failed: %v", err)
	}

	// Verify fact is now verified and partnership stats recorded
	updatedFact, found := k.GetFact(ctx, "fact-prov-partner")
	if !found {
		t.Fatal("fact not found")
	}
	if updatedFact.Status != types.FactStatusVerified {
		t.Errorf("expected verified status, got %s", updatedFact.Status)
	}
	if updatedFact.PartnershipId != "p-prov-test" {
		t.Errorf("expected partnership_id 'p-prov-test', got %q", updatedFact.PartnershipId)
	}

	stats := k.GetPartnershipCitationStats(ctx, "p-prov-test")
	if stats.TotalFactsCreated != 1 {
		t.Errorf("expected 1 fact in partnership stats, got %d", stats.TotalFactsCreated)
	}
}

func TestPartnershipAttribution_EvaluatedToVerified(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_eval_test_____")).String()
	agentAddr := sdk.AccAddress([]byte("agent_eval_test_____")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "p-eval-test",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Create an evaluated claim (score 300k-500k) with partnership
	claim := types.Claim{
		Id:                "claim-eval-partner",
		ClaimText:         "Evaluated test fact",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         agentAddr,
		StakeAmount:       "1000000",
		Status:            types.ClaimStatusEvaluated,
		StructuralScore:   400_000,
		ChallengeWindowEnd: 200,
		PartnershipId:     "p-eval-test",
		SubmittedAtHeight: 100,
	}
	k.SetClaim(ctx, &claim)

	// Transition to verified
	err := k.TransitionToVerified(ctx, &claim, 300)
	if err != nil {
		t.Fatalf("TransitionToVerified failed: %v", err)
	}

	// The fact created from evaluated claim should have partnership_id
	factID := claim.ProvisionalFactId // set by TransitionToVerified
	if factID == "" {
		t.Fatal("ProvisionalFactId not set after TransitionToVerified")
	}

	fact, found := k.GetFact(ctx, factID)
	if !found {
		t.Fatalf("fact %s not found", factID)
	}
	if fact.PartnershipId != "p-eval-test" {
		t.Errorf("expected partnership_id 'p-eval-test', got %q", fact.PartnershipId)
	}

	stats := k.GetPartnershipCitationStats(ctx, "p-eval-test")
	if stats.TotalFactsCreated != 1 {
		t.Errorf("expected 1 fact in partnership stats, got %d", stats.TotalFactsCreated)
	}
}
