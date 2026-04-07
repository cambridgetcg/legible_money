package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// TestYieldEmptyFarm verifies that ComputeFarmYield returns zero values when
// there are no facts in the store.
func TestYieldEmptyFarm(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fy := k.ComputeFarmYield(ctx)

	if fy.TotalFacts != 0 {
		t.Fatalf("expected TotalFacts=0, got %d", fy.TotalFacts)
	}
	if fy.TotalVerified != 0 {
		t.Fatalf("expected TotalVerified=0, got %d", fy.TotalVerified)
	}
	if fy.DomainCount != 0 {
		t.Fatalf("expected DomainCount=0, got %d", fy.DomainCount)
	}
	if fy.OverallDensity != 0 {
		t.Fatalf("expected OverallDensity=0, got %d", fy.OverallDensity)
	}
	if fy.BlockHeight != ctx.BlockHeight() {
		t.Fatalf("expected BlockHeight=%d, got %d", ctx.BlockHeight(), fy.BlockHeight)
	}
}

// TestYieldSingleDomain creates facts in one domain and verifies per-domain metrics.
func TestYieldSingleDomain(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create facts in the "physics" domain with varying statuses
	facts := []*types.Fact{
		{Id: "fact-1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000, CitationCount: 5, OriginalStake: "1000000"},
		{Id: "fact-2", Domain: "physics", Status: types.FactStatusVerified, Confidence: 900_000, CitationCount: 3, OriginalStake: "2000000"},
		{Id: "fact-3", Domain: "physics", Status: types.FactStatusProvisional, Confidence: 500_000, CitationCount: 0, OriginalStake: "500000"},
		{Id: "fact-4", Domain: "physics", Status: types.FactStatusDisproven, Confidence: 100_000, CitationCount: 0, OriginalStake: "100000"},
		{Id: "fact-5", Domain: "physics", Status: types.FactStatusActive, Confidence: 750_000, CitationCount: 2, OriginalStake: "3000000"},
	}
	for _, f := range facts {
		k.SetFact(ctx, f)
	}

	fy := k.ComputeFarmYield(ctx)

	if fy.DomainCount != 1 {
		t.Fatalf("expected DomainCount=1, got %d", fy.DomainCount)
	}
	if fy.TotalFacts != 5 {
		t.Fatalf("expected TotalFacts=5, got %d", fy.TotalFacts)
	}
	if fy.TotalVerified != 3 {
		t.Fatalf("expected TotalVerified=3 (2 verified + 1 active), got %d", fy.TotalVerified)
	}
	if fy.TotalCitations != 10 {
		t.Fatalf("expected TotalCitations=10, got %d", fy.TotalCitations)
	}

	// Find the physics domain
	var phys *keeper.DomainYield
	for i := range fy.Domains {
		if fy.Domains[i].Domain == "physics" {
			phys = &fy.Domains[i]
			break
		}
	}
	if phys == nil {
		t.Fatal("expected physics domain in yield")
	}

	if phys.TotalFacts != 5 {
		t.Fatalf("physics: expected TotalFacts=5, got %d", phys.TotalFacts)
	}
	if phys.VerifiedFacts != 3 {
		t.Fatalf("physics: expected VerifiedFacts=3, got %d", phys.VerifiedFacts)
	}
	if phys.ProvisionalFacts != 1 {
		t.Fatalf("physics: expected ProvisionalFacts=1, got %d", phys.ProvisionalFacts)
	}
	if phys.RejectedFacts != 1 {
		t.Fatalf("physics: expected RejectedFacts=1, got %d", phys.RejectedFacts)
	}
	if phys.TotalCitations != 10 {
		t.Fatalf("physics: expected TotalCitations=10, got %d", phys.TotalCitations)
	}

	// Avg confidence: (800k + 900k + 500k + 100k + 750k) / 5 = 610k
	expectedAvgConf := uint64(610_000)
	if phys.AvgConfidence != expectedAvgConf {
		t.Fatalf("physics: expected AvgConfidence=%d, got %d", expectedAvgConf, phys.AvgConfidence)
	}

	// Citation density: 10 * 1000 / 5 = 2000
	expectedDensity := uint64(2000)
	if phys.CitationDensity != expectedDensity {
		t.Fatalf("physics: expected CitationDensity=%d, got %d", expectedDensity, phys.CitationDensity)
	}

	// Stake: 1M + 2M + 500k + 100k + 3M = 6,600,000
	if phys.StakeInvested != "6600000" {
		t.Fatalf("physics: expected StakeInvested=6600000, got %s", phys.StakeInvested)
	}
}

// TestYieldMultiDomain creates facts across multiple domains and verifies
// that aggregation is correct.
func TestYieldMultiDomain(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	facts := []*types.Fact{
		// physics: 2 verified, 1 provisional
		{Id: "p1", Domain: "physics", Status: types.FactStatusVerified, Confidence: 900_000, CitationCount: 4, OriginalStake: "1000000"},
		{Id: "p2", Domain: "physics", Status: types.FactStatusVerified, Confidence: 800_000, CitationCount: 2, OriginalStake: "1000000"},
		{Id: "p3", Domain: "physics", Status: types.FactStatusProvisional, Confidence: 400_000, CitationCount: 0, OriginalStake: "500000"},

		// biology: 1 verified, 1 rejected
		{Id: "b1", Domain: "biology", Status: types.FactStatusActive, Confidence: 700_000, CitationCount: 1, OriginalStake: "2000000"},
		{Id: "b2", Domain: "biology", Status: types.FactStatusRevoked, Confidence: 0, CitationCount: 0, OriginalStake: "100000"},

		// mathematics: 1 verified, no citations
		{Id: "m1", Domain: "mathematics", Status: types.FactStatusVerified, Confidence: 1_000_000, CitationCount: 0, OriginalStake: "5000000"},
	}
	for _, f := range facts {
		k.SetFact(ctx, f)
	}

	fy := k.ComputeFarmYield(ctx)

	if fy.DomainCount != 3 {
		t.Fatalf("expected DomainCount=3, got %d", fy.DomainCount)
	}
	if fy.TotalFacts != 6 {
		t.Fatalf("expected TotalFacts=6, got %d", fy.TotalFacts)
	}
	if fy.TotalVerified != 4 {
		t.Fatalf("expected TotalVerified=4, got %d", fy.TotalVerified)
	}
	if fy.TotalCitations != 7 {
		t.Fatalf("expected TotalCitations=7, got %d", fy.TotalCitations)
	}

	// Overall density: 7 * 1000 / 6 = 1166
	expectedOverall := uint64(1166)
	if fy.OverallDensity != expectedOverall {
		t.Fatalf("expected OverallDensity=%d, got %d", expectedOverall, fy.OverallDensity)
	}

	// Verify per-domain
	domainMap := make(map[string]keeper.DomainYield)
	for _, d := range fy.Domains {
		domainMap[d.Domain] = d
	}

	// physics
	phys := domainMap["physics"]
	if phys.TotalFacts != 3 || phys.VerifiedFacts != 2 || phys.ProvisionalFacts != 1 {
		t.Fatalf("physics: unexpected counts: total=%d verified=%d provisional=%d",
			phys.TotalFacts, phys.VerifiedFacts, phys.ProvisionalFacts)
	}
	// Citation density: 6 * 1000 / 3 = 2000
	if phys.CitationDensity != 2000 {
		t.Fatalf("physics: expected CitationDensity=2000, got %d", phys.CitationDensity)
	}

	// biology
	bio := domainMap["biology"]
	if bio.TotalFacts != 2 || bio.VerifiedFacts != 1 || bio.RejectedFacts != 1 {
		t.Fatalf("biology: unexpected counts: total=%d verified=%d rejected=%d",
			bio.TotalFacts, bio.VerifiedFacts, bio.RejectedFacts)
	}

	// mathematics: all verified, zero citations
	math := domainMap["mathematics"]
	if math.TotalFacts != 1 || math.VerifiedFacts != 1 {
		t.Fatalf("mathematics: unexpected counts: total=%d verified=%d",
			math.TotalFacts, math.VerifiedFacts)
	}
	if math.CitationDensity != 0 {
		t.Fatalf("mathematics: expected CitationDensity=0, got %d", math.CitationDensity)
	}
	if math.AvgConfidence != 1_000_000 {
		t.Fatalf("mathematics: expected AvgConfidence=1000000, got %d", math.AvgConfidence)
	}
}

// TestYieldCitationDensityPrecision verifies that citation density calculation
// handles fractional results via the *1000 multiplier.
func TestYieldCitationDensityPrecision(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// 3 facts, 1 citation total: density = 1 * 1000 / 3 = 333
	facts := []*types.Fact{
		{Id: "f1", Domain: "logic", Status: types.FactStatusVerified, Confidence: 900_000, CitationCount: 1},
		{Id: "f2", Domain: "logic", Status: types.FactStatusVerified, Confidence: 900_000, CitationCount: 0},
		{Id: "f3", Domain: "logic", Status: types.FactStatusVerified, Confidence: 900_000, CitationCount: 0},
	}
	for _, f := range facts {
		k.SetFact(ctx, f)
	}

	fy := k.ComputeFarmYield(ctx)

	var logic *keeper.DomainYield
	for i := range fy.Domains {
		if fy.Domains[i].Domain == "logic" {
			logic = &fy.Domains[i]
			break
		}
	}
	if logic == nil {
		t.Fatal("expected logic domain")
	}

	// 1 * 1000 / 3 = 333 (integer division)
	if logic.CitationDensity != 333 {
		t.Fatalf("expected CitationDensity=333, got %d", logic.CitationDensity)
	}
}

// TestYieldStakeAccumulation verifies stake is correctly summed across facts.
func TestYieldStakeAccumulation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	facts := []*types.Fact{
		{Id: "s1", Domain: "economics", Status: types.FactStatusVerified, Confidence: 800_000, OriginalStake: "999999999"},
		{Id: "s2", Domain: "economics", Status: types.FactStatusVerified, Confidence: 800_000, OriginalStake: "1"},
		{Id: "s3", Domain: "economics", Status: types.FactStatusVerified, Confidence: 800_000, OriginalStake: ""},  // empty stake
		{Id: "s4", Domain: "economics", Status: types.FactStatusVerified, Confidence: 800_000, OriginalStake: "0"}, // zero stake
	}
	for _, f := range facts {
		k.SetFact(ctx, f)
	}

	fy := k.ComputeFarmYield(ctx)
	eco := fy.Domains[0]

	if eco.StakeInvested != "1000000000" {
		t.Fatalf("expected StakeInvested=1000000000, got %s", eco.StakeInvested)
	}
}

// TestYieldBlockHeight ensures the block height from context is propagated.
func TestYieldBlockHeight(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// The default setupKeeper sets height=10
	fy := k.ComputeFarmYield(ctx)
	if fy.BlockHeight != 10 {
		t.Fatalf("expected BlockHeight=10, got %d", fy.BlockHeight)
	}
}
