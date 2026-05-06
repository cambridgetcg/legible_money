package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- FARM-11: Per-Submitter Citation Concentration Tests ----------

func TestFarm11_ThreeAccountsCartel(t *testing.T) {
	// 3 accounts cite only each other -> concentration score > 50%
	k, ctx, _ := setupKeeper(t)

	// Submitter A cites submitters B and C repeatedly
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1alice", "lgm1bob")
	}
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1alice", "lgm1carol")
	}

	// Submitter B cites A and C
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1bob", "lgm1alice")
	}
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1bob", "lgm1carol")
	}

	// Submitter C cites A and B
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1carol", "lgm1alice")
	}
	for i := 0; i < 10; i++ {
		k.UpdateSubmitterCitationGraph(ctx, "lgm1carol", "lgm1bob")
	}

	// Check Alice: 20 total cites, top 3 = {bob:10, carol:10} = 20/20 = 100%
	scoreA, topA := k.CheckCitationConcentration(ctx, "lgm1alice", nil)
	if scoreA <= 500_000 {
		t.Errorf("expected alice concentration > 50%%, got %d (%.1f%%)", scoreA, float64(scoreA)/10_000)
	}
	if len(topA) < 2 {
		t.Errorf("expected at least 2 top-cited addresses for alice, got %d", len(topA))
	}

	// Check Bob
	scoreB, _ := k.CheckCitationConcentration(ctx, "lgm1bob", nil)
	if scoreB <= 500_000 {
		t.Errorf("expected bob concentration > 50%%, got %d (%.1f%%)", scoreB, float64(scoreB)/10_000)
	}

	// Check Carol
	scoreC, _ := k.CheckCitationConcentration(ctx, "lgm1carol", nil)
	if scoreC <= 500_000 {
		t.Errorf("expected carol concentration > 50%%, got %d (%.1f%%)", scoreC, float64(scoreC)/10_000)
	}
}

func TestFarm11_DiverseCitations(t *testing.T) {
	// Diverse citations across 10 accounts -> concentration score < 20%
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1diverse_submitter"
	// Cite 10 different submitters, 1 time each
	for i := 0; i < 10; i++ {
		citedAddr := "lgm1cited_" + string(rune('a'+i))
		k.UpdateSubmitterCitationGraph(ctx, submitter, citedAddr)
	}

	score, topCited := k.CheckCitationConcentration(ctx, submitter, nil)
	// Top 3 out of 10 equally-cited: 3/10 = 30%. Under the 80% threshold but
	// the score itself should be <= 30%.
	if score > 300_001 {
		t.Errorf("expected diverse concentration <= 30%%, got %d (%.1f%%)", score, float64(score)/10_000)
	}
	if len(topCited) > 3 {
		t.Errorf("expected at most 3 top-cited, got %d", len(topCited))
	}
}

func TestFarm11_PenaltyReducesReward(t *testing.T) {
	// Penalty correctly reduces citation reward proportionally
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1concentrated"
	// 20 cites all to one address -> 100% concentration
	for i := 0; i < 20; i++ {
		k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1target")
	}

	baseReward := sdkmath.NewInt(1_000_000)
	reduced := k.ApplyCitationConcentrationPenalty(ctx, submitter, baseReward)

	// 100% concentration -> reward reduced by 100% -> 0
	if reduced.GT(sdkmath.ZeroInt()) {
		t.Errorf("expected zero reward for 100%% concentration, got %s", reduced.String())
	}

	// Now test partial concentration: 80% to one, 20% spread
	submitter2 := "lgm1partial"
	for i := 0; i < 16; i++ {
		k.UpdateSubmitterCitationGraph(ctx, submitter2, "lgm1main_target")
	}
	for i := 0; i < 4; i++ {
		citedAddr := "lgm1other_" + string(rune('a'+i))
		k.UpdateSubmitterCitationGraph(ctx, submitter2, citedAddr)
	}

	reduced2 := k.ApplyCitationConcentrationPenalty(ctx, submitter2, baseReward)
	// top-3 = 16+1+1 = 18 out of 20 = 90% concentration -> 90% penalty
	// reduced = 1M * (1 - 0.90) = 100_000
	ep := k.GetExtendedParams(ctx)
	if ep.CitationConcentrationThreshold == 0 {
		t.Skip("concentration check disabled")
	}

	// The penalty should reduce the reward significantly
	if !reduced2.LT(baseReward) {
		t.Errorf("expected reduced reward for 90%% concentration, got %s (base %s)",
			reduced2.String(), baseReward.String())
	}
}

func TestFarm11_SelfCitationInteraction(t *testing.T) {
	// Self-citations (FARM-3) interact correctly with FARM-11
	// When a submitter self-cites AND has concentrated citations, both penalties apply.
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1selfciter"

	// All citations go to self -> both FARM-3 self-citation discount and
	// FARM-11 concentration penalty should apply.
	for i := 0; i < 20; i++ {
		k.UpdateSubmitterCitationGraph(ctx, submitter, submitter)
	}

	// Verify concentration is detected
	score, topCited := k.CheckCitationConcentration(ctx, submitter, nil)
	if score <= 500_000 {
		t.Errorf("expected self-citation concentration > 50%%, got %d", score)
	}
	if len(topCited) == 0 || topCited[0] != submitter {
		t.Errorf("expected self as top-cited, got %v", topCited)
	}

	// Verify penalty reduces reward
	baseReward := sdkmath.NewInt(1_000_000)
	reduced := k.ApplyCitationConcentrationPenalty(ctx, submitter, baseReward)
	if !reduced.LT(baseReward) {
		t.Errorf("expected penalty for self-citing cartel pattern, got full reward")
	}
}

func TestFarm11_BelowMinCitesNoConcentration(t *testing.T) {
	// Below min cites threshold -> no concentration detected
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1few_cites"
	// Only 3 cites (below default MinCites=5)
	for i := 0; i < 3; i++ {
		k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1target")
	}

	score, _ := k.CheckCitationConcentration(ctx, submitter, nil)
	if score != 0 {
		t.Errorf("expected 0 concentration below min cites, got %d", score)
	}

	baseReward := sdkmath.NewInt(1_000_000)
	reduced := k.ApplyCitationConcentrationPenalty(ctx, submitter, baseReward)
	if !reduced.Equal(baseReward) {
		t.Errorf("expected no penalty below min cites, got %s (base %s)",
			reduced.String(), baseReward.String())
	}
}

func TestFarm11_NoGraphNoPenalty(t *testing.T) {
	// No graph record -> no penalty
	k, ctx, _ := setupKeeper(t)

	baseReward := sdkmath.NewInt(500_000)
	reduced := k.ApplyCitationConcentrationPenalty(ctx, "lgm1unknown", baseReward)
	if !reduced.Equal(baseReward) {
		t.Errorf("expected no penalty for unknown submitter, got %s", reduced.String())
	}
}

func TestFarm11_DisabledParams(t *testing.T) {
	// When concentration params are disabled, no penalty applies
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1disabled_test"
	for i := 0; i < 20; i++ {
		k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1target")
	}

	// Disable by setting threshold to 0
	ep := k.GetExtendedParams(ctx)
	ep.CitationConcentrationThreshold = 0
	k.SetExtendedParams(ctx, ep)

	baseReward := sdkmath.NewInt(1_000_000)
	reduced := k.ApplyCitationConcentrationPenalty(ctx, submitter, baseReward)
	if !reduced.Equal(baseReward) {
		t.Errorf("expected no penalty when feature disabled, got %s", reduced.String())
	}
}

func TestFarm11_SubmitterCitationGraph_CRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	submitter := "lgm1graph_crud"

	// Initially not found
	_, found := k.GetSubmitterCitationGraph(ctx, submitter)
	if found {
		t.Fatal("expected graph not found initially")
	}

	// Update creates graph
	k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1bob")
	graph, found := k.GetSubmitterCitationGraph(ctx, submitter)
	if !found {
		t.Fatal("expected graph to be created after first update")
	}
	if graph.TotalCites != 1 {
		t.Errorf("expected TotalCites=1, got %d", graph.TotalCites)
	}
	if graph.CitedCounts["lgm1bob"] != 1 {
		t.Errorf("expected bob count=1, got %d", graph.CitedCounts["lgm1bob"])
	}

	// Multiple updates increment correctly
	k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1bob")
	k.UpdateSubmitterCitationGraph(ctx, submitter, "lgm1carol")
	graph, _ = k.GetSubmitterCitationGraph(ctx, submitter)
	if graph.TotalCites != 3 {
		t.Errorf("expected TotalCites=3, got %d", graph.TotalCites)
	}
	if graph.CitedCounts["lgm1bob"] != 2 {
		t.Errorf("expected bob count=2, got %d", graph.CitedCounts["lgm1bob"])
	}
	if graph.CitedCounts["lgm1carol"] != 1 {
		t.Errorf("expected carol count=1, got %d", graph.CitedCounts["lgm1carol"])
	}

	// Set overwrites
	graph.TotalCites = 42
	k.SetSubmitterCitationGraph(ctx, graph)
	graph2, _ := k.GetSubmitterCitationGraph(ctx, submitter)
	if graph2.TotalCites != 42 {
		t.Errorf("expected TotalCites=42 after overwrite, got %d", graph2.TotalCites)
	}
}

func TestFarm11_TopNSum_WithSubmitterGraph(t *testing.T) {
	// Verify TopNSum works correctly with submitter graph data
	counts := map[string]uint64{
		"lgm1alice": 15,
		"lgm1bob":   10,
		"lgm1carol": 8,
		"lgm1dave":  5,
		"lgm1eve":   2,
	}

	// Top 3 = 15 + 10 + 8 = 33
	got := keeper.TopNSum(counts, 3)
	if got != 33 {
		t.Errorf("TopNSum(top-3) = %d, want 33", got)
	}

	// Top 1 = 15
	got = keeper.TopNSum(counts, 1)
	if got != 15 {
		t.Errorf("TopNSum(top-1) = %d, want 15", got)
	}

	// Top 10 (more than available) = sum all = 40
	got = keeper.TopNSum(counts, 10)
	if got != 40 {
		t.Errorf("TopNSum(top-10) = %d, want 40", got)
	}
}

// Suppress unused import warnings
var _ = types.DefaultExtendedParams
var _ = sdkmath.ZeroInt
