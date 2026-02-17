package orchestrator

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// --- Navigator-specific mock QC ---

type navigatorTestQC struct {
	contextTestQC
}

func newNavigatorTestQC() *navigatorTestQC {
	return &navigatorTestQC{
		contextTestQC: contextTestQC{
			claims:       make(map[string]*agentsdk.ClaimInfo),
			facts:        make(map[string][]agentsdk.FactInfo),
			refs:         make(map[string][]agentsdk.FactReference),
			counterFacts: make(map[string][]agentsdk.CounterFactInfo),
		},
	}
}

// --- FollowReferences Tests ---

func TestFollowReferencesDepth1(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.refs["fact-a"] = []agentsdk.FactReference{
		{FromFactID: "fact-a", ToFactID: "fact-b", Type: "supports", Weight: 800_000},
		{FromFactID: "fact-a", ToFactID: "fact-c", Type: "extends", Weight: 500_000},
	}

	nav := NewDefaultKnowledgeNavigator(qc)
	subgraph, err := nav.FollowReferences(context.Background(), "fact-a", 1)
	if err != nil {
		t.Fatalf("FollowReferences: %v", err)
	}

	// Nodes: fact-a (visited)
	if len(subgraph.Nodes) != 1 {
		t.Fatalf("expected 1 node (just the start), got %d", len(subgraph.Nodes))
	}

	// Edges: 2 from fact-a
	if len(subgraph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(subgraph.Edges))
	}
}

func TestFollowReferencesDepth2(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.refs["fact-a"] = []agentsdk.FactReference{
		{FromFactID: "fact-a", ToFactID: "fact-b", Type: "supports", Weight: 800_000},
	}
	qc.refs["fact-b"] = []agentsdk.FactReference{
		{FromFactID: "fact-b", ToFactID: "fact-c", Type: "supports", Weight: 700_000},
	}

	nav := NewDefaultKnowledgeNavigator(qc)
	subgraph, err := nav.FollowReferences(context.Background(), "fact-a", 2)
	if err != nil {
		t.Fatalf("FollowReferences: %v", err)
	}

	// Should visit fact-a at depth 0, fact-b at depth 1
	if len(subgraph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(subgraph.Nodes))
	}

	// Edges: 1 from fact-a, 1 from fact-b = 2
	if len(subgraph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(subgraph.Edges))
	}
}

func TestFollowReferencesSortedDeterministic(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.refs["fact-root"] = []agentsdk.FactReference{
		{FromFactID: "fact-root", ToFactID: "fact-z", Type: "supports", Weight: 500_000},
		{FromFactID: "fact-root", ToFactID: "fact-a", Type: "supports", Weight: 600_000},
	}

	nav := NewDefaultKnowledgeNavigator(qc)
	subgraph, err := nav.FollowReferences(context.Background(), "fact-root", 1)
	if err != nil {
		t.Fatalf("FollowReferences: %v", err)
	}

	// Nodes should be sorted by ID
	if len(subgraph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(subgraph.Nodes))
	}
	if subgraph.Nodes[0].ID != "fact-root" {
		t.Fatalf("expected fact-root, got %s", subgraph.Nodes[0].ID)
	}

	// Edges sorted by FromFactID then ToFactID
	if subgraph.Edges[0].ToFactID != "fact-a" {
		t.Fatalf("expected edge to fact-a first, got %s", subgraph.Edges[0].ToFactID)
	}
}

func TestFollowReferencesNoCycle(t *testing.T) {
	qc := newNavigatorTestQC()
	// Circular reference: a -> b -> a
	qc.refs["fact-a"] = []agentsdk.FactReference{
		{FromFactID: "fact-a", ToFactID: "fact-b", Type: "supports", Weight: 500_000},
	}
	qc.refs["fact-b"] = []agentsdk.FactReference{
		{FromFactID: "fact-b", ToFactID: "fact-a", Type: "supports", Weight: 500_000},
	}

	nav := NewDefaultKnowledgeNavigator(qc)
	subgraph, err := nav.FollowReferences(context.Background(), "fact-a", 5)
	if err != nil {
		t.Fatalf("FollowReferences: %v", err)
	}

	// Should not infinite loop — both visited
	if len(subgraph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes (no infinite cycle), got %d", len(subgraph.Nodes))
	}
}

// --- ScoreNovelty Tests ---

func TestScoreNoveltyEmptyDomain(t *testing.T) {
	nav := NewDefaultKnowledgeNavigator(nil)
	claim := &agentsdk.ClaimInfo{Text: "something new"}

	score := nav.ScoreNovelty(context.Background(), claim, nil)
	if score != 10000 {
		t.Fatalf("expected 10000 for empty domain, got %d", score)
	}
}

func TestScoreNoveltyDuplicate(t *testing.T) {
	nav := NewDefaultKnowledgeNavigator(nil)
	claim := &agentsdk.ClaimInfo{Text: "water is wet"}
	facts := []agentsdk.FactInfo{
		{ID: "f1", ClaimText: "water is wet"},
	}

	score := nav.ScoreNovelty(context.Background(), claim, facts)
	if score != 0 {
		t.Fatalf("expected 0 for duplicate, got %d", score)
	}
}

func TestScoreNoveltyDensePenalty(t *testing.T) {
	nav := NewDefaultKnowledgeNavigator(nil)
	claim := &agentsdk.ClaimInfo{Text: "a unique claim"}

	// 50 existing facts → penalty = 50 * 100 = 5000 → novelty = 5000
	facts := make([]agentsdk.FactInfo, 50)
	for i := range facts {
		facts[i] = agentsdk.FactInfo{ID: "f", ClaimText: "other text"}
	}

	score := nav.ScoreNovelty(context.Background(), claim, facts)
	if score != 5000 {
		t.Fatalf("expected 5000, got %d", score)
	}
}

func TestScoreNoveltyMaxPenalty(t *testing.T) {
	nav := NewDefaultKnowledgeNavigator(nil)
	claim := &agentsdk.ClaimInfo{Text: "unique"}

	// 100 facts → penalty capped at 8000 → novelty = 2000
	facts := make([]agentsdk.FactInfo, 100)
	for i := range facts {
		facts[i] = agentsdk.FactInfo{ID: "f", ClaimText: "other"}
	}

	score := nav.ScoreNovelty(context.Background(), claim, facts)
	if score != 2000 {
		t.Fatalf("expected 2000 (capped penalty), got %d", score)
	}
}

func TestScoreNoveltyNilClaim(t *testing.T) {
	nav := NewDefaultKnowledgeNavigator(nil)
	score := nav.ScoreNovelty(context.Background(), nil, nil)
	if score != 0 {
		t.Fatalf("expected 0 for nil claim, got %d", score)
	}
}

// --- FindCitationChain Tests ---

func TestFindCitationChainSimple(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.refs["fact-a"] = []agentsdk.FactReference{
		{FromFactID: "fact-a", ToFactID: "fact-b", Type: "supports", Weight: 800_000},
	}
	// fact-b has no supporting refs — it's a foundation
	qc.refs["fact-b"] = nil

	nav := NewDefaultKnowledgeNavigator(qc)
	chain, err := nav.FindCitationChain(context.Background(), "fact-a")
	if err != nil {
		t.Fatalf("FindCitationChain: %v", err)
	}

	if chain.FactID != "fact-a" {
		t.Fatalf("expected fact-a, got %s", chain.FactID)
	}

	// Should have 1 ancestor level with fact-b
	if len(chain.Ancestors) != 1 {
		t.Fatalf("expected 1 ancestor level, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].Facts[0].ID != "fact-b" {
		t.Fatalf("expected fact-b at level 0, got %s", chain.Ancestors[0].Facts[0].ID)
	}
}

func TestFindCitationChainFoundation(t *testing.T) {
	qc := newNavigatorTestQC()
	// a -> b -> c (foundation)
	qc.refs["fact-a"] = []agentsdk.FactReference{
		{FromFactID: "fact-a", ToFactID: "fact-b", Type: "supports", Weight: 800_000},
	}
	qc.refs["fact-b"] = []agentsdk.FactReference{
		{FromFactID: "fact-b", ToFactID: "fact-c", Type: "supports", Weight: 700_000},
	}
	qc.refs["fact-c"] = nil // foundation — no supporting refs

	nav := NewDefaultKnowledgeNavigator(qc)
	chain, err := nav.FindCitationChain(context.Background(), "fact-a")
	if err != nil {
		t.Fatalf("FindCitationChain: %v", err)
	}

	// fact-c is the foundation: it has no outgoing supporting refs.
	// fact-b is NOT foundation because it has a supporting ref to fact-c.
	foundationMap := make(map[string]bool)
	for _, f := range chain.Foundation {
		foundationMap[f] = true
	}
	if !foundationMap["fact-c"] {
		t.Fatalf("expected fact-c in foundation (leaf with no supporting refs), got %v", chain.Foundation)
	}
	if foundationMap["fact-b"] {
		t.Fatal("fact-b should NOT be in foundation — it has a supporting ref to fact-c")
	}
}

func TestFindCitationChainSortedFoundation(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.refs["fact-root"] = []agentsdk.FactReference{
		{FromFactID: "fact-root", ToFactID: "fact-z", Type: "supports", Weight: 500_000},
		{FromFactID: "fact-root", ToFactID: "fact-a", Type: "supports", Weight: 600_000},
	}

	nav := NewDefaultKnowledgeNavigator(qc)
	chain, err := nav.FindCitationChain(context.Background(), "fact-root")
	if err != nil {
		t.Fatalf("FindCitationChain: %v", err)
	}

	// Foundation should be sorted
	if len(chain.Foundation) >= 2 {
		if chain.Foundation[0] > chain.Foundation[1] {
			t.Fatalf("foundation not sorted: %v", chain.Foundation)
		}
	}
}

// --- GapDetector Tests ---

func TestGapDetectorLowDensity(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.facts["sparse"] = []agentsdk.FactInfo{
		{ID: "f1", Confidence: 600_000, Category: "formal"},
	}

	detector := NewDefaultGapDetector(qc)
	gaps, err := detector.DetectGaps(context.Background(), "sparse")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}

	found := false
	for _, g := range gaps {
		if g.Type == "low_density" {
			found = true
			// Score = (10 - 1) * 1000 = 9000
			if g.Score != 9000 {
				t.Fatalf("expected low_density score 9000, got %d", g.Score)
			}
		}
	}
	if !found {
		t.Fatal("expected low_density gap")
	}
}

func TestGapDetectorDecaying(t *testing.T) {
	qc := newNavigatorTestQC()
	facts := make([]agentsdk.FactInfo, 15)
	for i := range facts {
		facts[i] = agentsdk.FactInfo{
			ID:         "f" + string(rune('a'+i)),
			Confidence: 200_000, // all below threshold
			Category:   "empirical",
		}
	}
	qc.facts["decaying"] = facts

	detector := NewDefaultGapDetector(qc)
	gaps, err := detector.DetectGaps(context.Background(), "decaying")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}

	found := false
	for _, g := range gaps {
		if g.Type == "decaying" {
			found = true
			if len(g.RelatedFacts) != 15 {
				t.Fatalf("expected 15 decaying facts, got %d", len(g.RelatedFacts))
			}
		}
	}
	if !found {
		t.Fatal("expected decaying gap")
	}
}

func TestGapDetectorUnchallenged(t *testing.T) {
	qc := newNavigatorTestQC()
	qc.facts["unchallenged"] = []agentsdk.FactInfo{
		{ID: "f-strong", Confidence: 900_000, Category: "protocol"},
	}
	// No counter-facts for f-strong

	detector := &DefaultGapDetector{
		qc:             qc,
		minDensity:     1, // low threshold to avoid low_density gap
		decayThreshold: 400_000,
	}
	gaps, err := detector.DetectGaps(context.Background(), "unchallenged")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}

	found := false
	for _, g := range gaps {
		if g.Type == "unchallenged" {
			found = true
			if len(g.RelatedFacts) != 1 {
				t.Fatalf("expected 1 unchallenged fact, got %d", len(g.RelatedFacts))
			}
			if g.RelatedFacts[0] != "f-strong" {
				t.Fatalf("expected f-strong, got %s", g.RelatedFacts[0])
			}
		}
	}
	if !found {
		t.Fatal("expected unchallenged gap")
	}
}

func TestGapDetectorSortedByScore(t *testing.T) {
	qc := newNavigatorTestQC()
	// 3 facts: 2 decaying, 1 strong
	qc.facts["mixed"] = []agentsdk.FactInfo{
		{ID: "f-decay-1", Confidence: 200_000, Category: "social"},
		{ID: "f-decay-2", Confidence: 300_000, Category: "social"},
		{ID: "f-strong", Confidence: 900_000, Category: "protocol"},
	}

	detector := &DefaultGapDetector{
		qc:             qc,
		minDensity:     5, // will trigger low_density (3 < 5)
		decayThreshold: 400_000,
	}
	gaps, err := detector.DetectGaps(context.Background(), "mixed")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}

	// Gaps should be sorted by score descending
	for i := 1; i < len(gaps); i++ {
		if gaps[i].Score > gaps[i-1].Score {
			t.Fatalf("gaps not sorted by score desc: gap[%d].Score=%d > gap[%d].Score=%d",
				i, gaps[i].Score, i-1, gaps[i-1].Score)
		}
	}
}

func TestGapDetectorEmptyDomain(t *testing.T) {
	qc := newNavigatorTestQC()

	detector := NewDefaultGapDetector(qc)
	gaps, err := detector.DetectGaps(context.Background(), "empty")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}

	// Should have low_density gap only (0 facts < 10 threshold)
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap (low_density), got %d", len(gaps))
	}
	if gaps[0].Type != "low_density" {
		t.Fatalf("expected low_density, got %s", gaps[0].Type)
	}
}

// --- Mock Tests ---

func TestMockKnowledgeNavigator(t *testing.T) {
	mock := NewMockKnowledgeNavigator()
	mock.Subgraphs["fact-1"] = &ReferenceSubgraph{
		Nodes: []agentsdk.FactInfo{{ID: "fact-1"}},
		Edges: nil,
		Depth: 1,
	}
	mock.NoveltyVal = 7500

	sg, err := mock.FollowReferences(context.Background(), "fact-1", 1)
	if err != nil {
		t.Fatalf("FollowReferences: %v", err)
	}
	if len(sg.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(sg.Nodes))
	}

	novelty := mock.ScoreNovelty(context.Background(), nil, nil)
	if novelty != 7500 {
		t.Fatalf("expected 7500, got %d", novelty)
	}
}

func TestMockGapDetector(t *testing.T) {
	mock := NewMockGapDetector()
	mock.Gaps["test"] = []KnowledgeGap{
		{Type: "low_density", Domain: "test", Score: 5000},
	}

	gaps, err := mock.DetectGaps(context.Background(), "test")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}

	// Empty domain returns nil
	gaps, err = mock.DetectGaps(context.Background(), "missing")
	if err != nil {
		t.Fatalf("DetectGaps: %v", err)
	}
	if gaps != nil {
		t.Fatalf("expected nil for missing domain, got %v", gaps)
	}
}

func TestDedupSorted(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "a", "b", "c", "c"}, []string{"a", "b", "c"}},
		{[]string{"x"}, []string{"x"}},
		{nil, nil},
		{[]string{}, []string{}},
	}

	for _, tt := range tests {
		result := dedup(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("dedup(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
