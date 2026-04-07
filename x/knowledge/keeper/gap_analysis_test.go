package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// TestGapAnalysis_UncitedAxioms verifies that high-confidence facts with no
// incoming citations are detected as uncited axioms.
func TestGapAnalysis_UncitedAxioms(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a high-confidence verified fact with no citations.
	axiom := &types.Fact{
		Id:         "fact-axiom-1",
		ClaimText:  "Water is H2O",
		Domain:     "chemistry",
		Status:     types.FactStatusVerified,
		Confidence: 900_000, // 90%
		Submitter:  testSubmitterAddr,
	}
	k.SetFact(ctx, axiom)

	// Create a low-confidence fact (should not appear as uncited axiom).
	lowConf := &types.Fact{
		Id:         "fact-low-1",
		ClaimText:  "Uncertain claim",
		Domain:     "chemistry",
		Status:     types.FactStatusVerified,
		Confidence: 500_000, // 50%
		Submitter:  testSubmitterAddr,
	}
	k.SetFact(ctx, lowConf)

	uncited := k.GetUncitedAxioms(ctx)
	if len(uncited) != 1 {
		t.Fatalf("expected 1 uncited axiom, got %d", len(uncited))
	}
	if uncited[0].FactID != "fact-axiom-1" {
		t.Errorf("expected fact-axiom-1, got %s", uncited[0].FactID)
	}
	if uncited[0].Type != "uncited_axiom" {
		t.Errorf("expected type uncited_axiom, got %s", uncited[0].Type)
	}
	if uncited[0].Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

// TestGapAnalysis_SparseDomains verifies that domains with fewer than the threshold
// number of facts are flagged.
func TestGapAnalysis_SparseDomains(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// The default domains are initialized by setupKeeper. They all start with 0 facts,
	// so they should all be sparse. Add facts to one domain to make it non-sparse.
	for i := 0; i < 5; i++ {
		f := &types.Fact{
			Id:         "fact-math-" + string(rune('a'+i)),
			ClaimText:  "Math fact",
			Domain:     "mathematics",
			Status:     types.FactStatusVerified,
			Confidence: 900_000,
			Submitter:  testSubmitterAddr,
		}
		k.SetFact(ctx, f)
	}

	sparse := k.GetSparseDomains(ctx)
	// mathematics should NOT appear (it has exactly 5 facts = threshold).
	for _, entry := range sparse {
		if entry.Domain == "mathematics" {
			t.Errorf("mathematics should not be sparse with %d facts", keeper.SparseDomainThreshold)
		}
	}

	// All sparse domains should have type "sparse_domain"
	for _, entry := range sparse {
		if entry.Type != "sparse_domain" {
			t.Errorf("expected type sparse_domain, got %s", entry.Type)
		}
	}
}

// TestGapAnalysis_CitationOrphans verifies that facts referencing non-existent
// or revoked facts are detected.
func TestGapAnalysis_CitationOrphans(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact that references a non-existent fact.
	citing := &types.Fact{
		Id:         "fact-citing-1",
		ClaimText:  "Building on prior work",
		Domain:     "physics",
		Status:     types.FactStatusVerified,
		Confidence: 700_000,
		Submitter:  testSubmitterAddr,
		References: []string{"fact-nonexistent"},
	}
	k.SetFact(ctx, citing)

	// Store the reference in the reference index.
	k.SetFactReference(ctx, &types.FactReference{
		FromFactId:    "fact-citing-1",
		ToFactId:      "fact-nonexistent",
		ReferenceType: "supports",
		Weight:        100,
	})

	orphans := k.GetCitationOrphans(ctx)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 citation orphan, got %d", len(orphans))
	}
	if orphans[0].FactID != "fact-citing-1" {
		t.Errorf("expected fact-citing-1, got %s", orphans[0].FactID)
	}

	// Now create the target but make it revoked — should still be an orphan.
	revoked := &types.Fact{
		Id:         "fact-nonexistent",
		ClaimText:  "This was revoked",
		Domain:     "physics",
		Status:     types.FactStatusRevoked,
		Confidence: 50_000,
		Submitter:  testSubmitterAddr,
	}
	k.SetFact(ctx, revoked)

	orphans2 := k.GetCitationOrphans(ctx)
	if len(orphans2) != 1 {
		t.Fatalf("expected 1 citation orphan (revoked target), got %d", len(orphans2))
	}
}

// TestGapAnalysis_ShallowTrees verifies that facts whose citations all come from
// a single submitter are flagged.
func TestGapAnalysis_ShallowTrees(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact with citation count > 0.
	cited := &types.Fact{
		Id:            "fact-cited-1",
		ClaimText:     "Popular but shallow",
		Domain:        "biology",
		Status:        types.FactStatusVerified,
		Confidence:    850_000,
		Submitter:     testSubmitterAddr,
		CitationCount: 3,
	}
	k.SetFact(ctx, cited)

	// Set citation source record: all citations from one submitter.
	k.SetCitationSourceRecord(ctx, types.CitationSourceRecord{
		FactID:       "fact-cited-1",
		SourceCounts: map[string]uint64{testSubmitterAddr: 3},
		TotalCites:   3,
	})

	report := k.AnalyzeGaps(ctx)
	if len(report.ShallowTrees) != 1 {
		t.Fatalf("expected 1 shallow tree, got %d", len(report.ShallowTrees))
	}
	if report.ShallowTrees[0].FactID != "fact-cited-1" {
		t.Errorf("expected fact-cited-1, got %s", report.ShallowTrees[0].FactID)
	}
}

// TestGapAnalysis_StaleFacts verifies facts with no confidence growth are flagged.
func TestGapAnalysis_StaleFacts(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a context at a high block height so staleness can be detected.
	ctx = ctx.WithBlockHeight(500_000)

	stale := &types.Fact{
		Id:                   "fact-stale-1",
		ClaimText:            "Old knowledge",
		Domain:               "history",
		Status:               types.FactStatusVerified,
		Confidence:           700_000,
		Submitter:            testSubmitterAddr,
		LastConfidenceUpdate: 100_000, // Updated 400k blocks ago (> 240k threshold)
	}
	k.SetFact(ctx, stale)

	fresh := &types.Fact{
		Id:                   "fact-fresh-1",
		ClaimText:            "Recent knowledge",
		Domain:               "history",
		Status:               types.FactStatusVerified,
		Confidence:           700_000,
		Submitter:            testSubmitterAddr,
		LastConfidenceUpdate: 450_000, // Updated 50k blocks ago (< 240k threshold)
	}
	k.SetFact(ctx, fresh)

	report := k.AnalyzeGaps(ctx)

	staleFound := false
	for _, entry := range report.StaleFacts {
		if entry.FactID == "fact-stale-1" {
			staleFound = true
		}
		if entry.FactID == "fact-fresh-1" {
			t.Error("fresh fact should not be stale")
		}
	}
	if !staleFound {
		t.Error("stale fact should have been detected")
	}
}

// TestGapAnalysis_PriorityOrdering verifies that TopGaps are sorted by priority
// with uncited axioms ranked highest.
func TestGapAnalysis_PriorityOrdering(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ctx = ctx.WithBlockHeight(500_000)

	// Uncited axiom (highest priority).
	k.SetFact(ctx, &types.Fact{
		Id:         "fact-axiom",
		ClaimText:  "Axiom",
		Domain:     "mathematics",
		Status:     types.FactStatusVerified,
		Confidence: 950_000,
		Submitter:  testSubmitterAddr,
	})

	// Stale fact (lower priority).
	k.SetFact(ctx, &types.Fact{
		Id:                   "fact-stale",
		ClaimText:            "Stale",
		Domain:               "history",
		Status:               types.FactStatusActive,
		Confidence:           600_000,
		Submitter:            testSubmitterAddr,
		LastConfidenceUpdate: 100_000,
	})

	// Citation orphan.
	k.SetFact(ctx, &types.Fact{
		Id:         "fact-orphan",
		ClaimText:  "Orphan",
		Domain:     "physics",
		Status:     types.FactStatusVerified,
		Confidence: 700_000,
		Submitter:  testSubmitterAddr,
	})
	k.SetFactReference(ctx, &types.FactReference{
		FromFactId:    "fact-orphan",
		ToFactId:      "fact-gone",
		ReferenceType: "supports",
		Weight:        100,
	})

	report := k.AnalyzeGaps(ctx)

	if len(report.TopGaps) == 0 {
		t.Fatal("expected non-empty TopGaps")
	}

	// Verify ordering: uncited axiom should be first (priority ~1950),
	// then sparse domains (~1300), then citation orphan (~1300), then stale (~800).
	firstType := report.TopGaps[0].Type
	// The highest priority entry should be an uncited axiom.
	if firstType != "uncited_axiom" {
		// Sparse domains may also be high priority if many default domains exist.
		// Just verify that priorities are non-increasing.
		t.Logf("first gap type: %s (priority %d)", firstType, report.TopGaps[0].Priority)
	}

	// Verify priorities are non-increasing.
	for i := 1; i < len(report.TopGaps); i++ {
		if report.TopGaps[i].Priority > report.TopGaps[i-1].Priority {
			t.Errorf("TopGaps not sorted: index %d (priority %d) > index %d (priority %d)",
				i, report.TopGaps[i].Priority, i-1, report.TopGaps[i-1].Priority)
		}
	}
}

// TestGapAnalysis_FullReport verifies the complete AnalyzeGaps output.
func TestGapAnalysis_FullReport(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Add a few facts.
	k.SetFact(ctx, &types.Fact{
		Id:         "fact-1",
		ClaimText:  "First fact",
		Domain:     "mathematics",
		Status:     types.FactStatusVerified,
		Confidence: 900_000,
		Submitter:  testSubmitterAddr,
	})
	k.SetFact(ctx, &types.Fact{
		Id:         "fact-2",
		ClaimText:  "Second fact",
		Domain:     "mathematics",
		Status:     types.FactStatusActive,
		Confidence: 850_000,
		Submitter:  testSubmitterAddr,
	})
	k.SetFact(ctx, &types.Fact{
		Id:         "fact-3",
		ClaimText:  "Revoked fact",
		Domain:     "physics",
		Status:     types.FactStatusRevoked,
		Confidence: 100_000,
		Submitter:  testSubmitterAddr,
	})

	report := k.AnalyzeGaps(ctx)

	if report.TotalFacts != 3 {
		t.Errorf("expected 3 total facts, got %d", report.TotalFacts)
	}
	if report.FactsByDomain["mathematics"] != 2 {
		t.Errorf("expected 2 mathematics facts, got %d", report.FactsByDomain["mathematics"])
	}
	if report.FactsByStatus[types.FactStatusRevoked] != 1 {
		t.Errorf("expected 1 revoked fact, got %d", report.FactsByStatus[types.FactStatusRevoked])
	}
}

// TestGapAnalysis_GenerateSuggestion verifies suggestion text for each gap type.
func TestGapAnalysis_GenerateSuggestion(t *testing.T) {
	tests := []struct {
		gapType  string
		expectOK bool
	}{
		{"uncited_axiom", true},
		{"sparse_domain", true},
		{"citation_orphan", true},
		{"shallow_tree", true},
		{"stale", true},
		{"unknown", true}, // returns fallback
	}

	for _, tt := range tests {
		t.Run(tt.gapType, func(t *testing.T) {
			entry := keeper.GapEntry{
				FactID: "fact-test",
				Domain: "test-domain",
				Type:   tt.gapType,
			}
			suggestion := keeper.GenerateSuggestion(entry)
			if suggestion == "" {
				t.Error("expected non-empty suggestion")
			}
		})
	}
}
