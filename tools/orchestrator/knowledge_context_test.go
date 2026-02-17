package orchestrator

import (
	"context"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// --- MockQueryClient for context tests ---

type contextTestQC struct {
	claims      map[string]*agentsdk.ClaimInfo
	facts       map[string][]agentsdk.FactInfo
	refs        map[string][]agentsdk.FactReference
	counterFacts map[string][]agentsdk.CounterFactInfo
}

func newContextTestQC() *contextTestQC {
	return &contextTestQC{
		claims:       make(map[string]*agentsdk.ClaimInfo),
		facts:        make(map[string][]agentsdk.FactInfo),
		refs:         make(map[string][]agentsdk.FactReference),
		counterFacts: make(map[string][]agentsdk.CounterFactInfo),
	}
}

func (q *contextTestQC) ChainStatus(_ context.Context) (*agentsdk.ChainStatus, error) {
	return &agentsdk.ChainStatus{LatestHeight: 100}, nil
}
func (q *contextTestQC) Balance(_ context.Context, _ string) (uint64, error) { return 0, nil }
func (q *contextTestQC) AgentProfile(_ context.Context, _ string) (*agentsdk.AgentProfileInfo, error) {
	return nil, nil
}
func (q *contextTestQC) ActiveRounds(_ context.Context) ([]agentsdk.RoundInfo, error) { return nil, nil }
func (q *contextTestQC) RoundDetail(_ context.Context, _ string) (*agentsdk.RoundInfo, error) {
	return nil, nil
}
func (q *contextTestQC) ClaimsBySubmitter(_ context.Context, _ string) ([]agentsdk.ClaimInfo, error) {
	return nil, nil
}
func (q *contextTestQC) GetClaim(_ context.Context, claimID string) (*agentsdk.ClaimInfo, error) {
	if c, ok := q.claims[claimID]; ok {
		return c, nil
	}
	return nil, nil
}
func (q *contextTestQC) FactsByDomain(_ context.Context, domain string) ([]agentsdk.FactInfo, error) {
	return q.facts[domain], nil
}
func (q *contextTestQC) FactReferences(_ context.Context, factID string) ([]agentsdk.FactReference, error) {
	return q.refs[factID], nil
}
func (q *contextTestQC) CounterFacts(_ context.Context, factID string) ([]agentsdk.CounterFactInfo, error) {
	return q.counterFacts[factID], nil
}

// --- Tests ---

func TestBuildContextBasic(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID:       "claim-1",
		Text:     "Water boils at 100 degrees Celsius",
		Domain:   "physics",
		Category: "empirical",
		Status:   "pending",
	}
	qc.facts["physics"] = []agentsdk.FactInfo{
		{ID: "fact-1", ClaimText: "Existing physics fact", Domain: "physics", Category: "empirical", Confidence: 700_000},
		{ID: "fact-2", ClaimText: "Another fact", Domain: "physics", Category: "formal", Confidence: 500_000},
	}

	provider := NewQueryClientContextProvider(qc)
	ctx := context.Background()

	kctx, err := provider.BuildContext(ctx, "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if kctx.Claim.ID != "claim-1" {
		t.Fatalf("expected claim-1, got %s", kctx.Claim.ID)
	}
	if kctx.Domain.FactCount != 2 {
		t.Fatalf("expected 2 domain facts, got %d", kctx.Domain.FactCount)
	}
	if kctx.DomainDensity != 2 {
		t.Fatalf("expected domain density 2, got %d", kctx.DomainDensity)
	}
}

func TestBuildContextDomainMeanConfidence(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID: "claim-1", Domain: "math", Category: "formal",
	}
	qc.facts["math"] = []agentsdk.FactInfo{
		{ID: "f1", Confidence: 600_000, Category: "formal"},
		{ID: "f2", Confidence: 800_000, Category: "formal"},
	}

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	// Mean = (600000 + 800000) / 2 = 700000
	if kctx.Domain.MeanConfBps != 700_000 {
		t.Fatalf("expected mean 700000, got %d", kctx.Domain.MeanConfBps)
	}
}

func TestBuildContextEmptyDomain(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID: "claim-1", Domain: "empty", Category: "formal",
	}
	// No facts in "empty" domain

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if kctx.Domain.FactCount != 0 {
		t.Fatalf("expected 0 facts in empty domain, got %d", kctx.Domain.FactCount)
	}
	if kctx.Domain.MeanConfBps != 0 {
		t.Fatalf("expected 0 mean confidence, got %d", kctx.Domain.MeanConfBps)
	}
}

func TestBuildContextStratumCeiling(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-social"] = &agentsdk.ClaimInfo{
		ID: "claim-social", Domain: "social", Category: "social",
	}

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-social")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	// Social domain stratum ceiling = 850_000
	if kctx.StratumCeiling != 850_000 {
		t.Fatalf("expected stratum ceiling 850000, got %d", kctx.StratumCeiling)
	}
}

func TestBuildContextCounterFacts(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID: "claim-1", Domain: "general", Category: "empirical",
	}
	qc.counterFacts["claim-1"] = []agentsdk.CounterFactInfo{
		{ID: "cf-1", OriginalFactID: "claim-1", NegationType: "contradiction", Confidence: 500_000},
	}

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if len(kctx.CounterFacts) != 1 {
		t.Fatalf("expected 1 counter-fact, got %d", len(kctx.CounterFacts))
	}
	if kctx.CounterFacts[0].NegationType != "contradiction" {
		t.Fatalf("expected contradiction, got %s", kctx.CounterFacts[0].NegationType)
	}
}

func TestBuildContextTopFactsSorted(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID: "claim-1", Domain: "test", Category: "formal",
	}
	qc.facts["test"] = []agentsdk.FactInfo{
		{ID: "f-low", Confidence: 100_000},
		{ID: "f-high", Confidence: 900_000},
		{ID: "f-mid", Confidence: 500_000},
	}

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	// TopFacts should be sorted by confidence desc
	if len(kctx.Domain.TopFacts) != 3 {
		t.Fatalf("expected 3 top facts, got %d", len(kctx.Domain.TopFacts))
	}
	if kctx.Domain.TopFacts[0].ID != "f-high" {
		t.Fatalf("expected f-high first, got %s", kctx.Domain.TopFacts[0].ID)
	}
	if kctx.Domain.TopFacts[2].ID != "f-low" {
		t.Fatalf("expected f-low last, got %s", kctx.Domain.TopFacts[2].ID)
	}
}

func TestBuildContextCategoryAggregation(t *testing.T) {
	qc := newContextTestQC()
	qc.claims["claim-1"] = &agentsdk.ClaimInfo{
		ID: "claim-1", Domain: "mixed", Category: "formal",
	}
	qc.facts["mixed"] = []agentsdk.FactInfo{
		{ID: "f1", Category: "formal", Confidence: 600_000},
		{ID: "f2", Category: "empirical", Confidence: 500_000},
		{ID: "f3", Category: "formal", Confidence: 700_000},
	}

	provider := NewQueryClientContextProvider(qc)
	kctx, err := provider.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if kctx.Domain.Categories["formal"] != 2 {
		t.Fatalf("expected 2 formal facts, got %d", kctx.Domain.Categories["formal"])
	}
	if kctx.Domain.Categories["empirical"] != 1 {
		t.Fatalf("expected 1 empirical fact, got %d", kctx.Domain.Categories["empirical"])
	}
}

func TestLookupStratumCeiling(t *testing.T) {
	tests := []struct {
		domain   string
		expected uint32
	}{
		{"fundamental", 1_000_000},
		{"social", 850_000},
		{"biological", 950_000},
		{"unknown_domain", defaultStratumCeiling},
	}

	for _, tt := range tests {
		got := LookupStratumCeiling(tt.domain)
		if got != tt.expected {
			t.Errorf("LookupStratumCeiling(%s) = %d, want %d", tt.domain, got, tt.expected)
		}
	}
}

func TestLookupLogicZoneCap(t *testing.T) {
	tests := []struct {
		zone     string
		expected uint32
	}{
		{"propositional", 1_000_000},
		{"peano", 850_000},
		{"empirical", 700_000},
		{"unknown_zone", defaultLogicZoneCap},
	}

	for _, tt := range tests {
		got := LookupLogicZoneCap(tt.zone)
		if got != tt.expected {
			t.Errorf("LookupLogicZoneCap(%s) = %d, want %d", tt.zone, got, tt.expected)
		}
	}
}

func TestMockKnowledgeContextProvider(t *testing.T) {
	mock := NewMockKnowledgeContextProvider()
	mock.AddContext("claim-1", &KnowledgeContext{
		Claim:         &agentsdk.ClaimInfo{ID: "claim-1"},
		StratumCeiling: 850_000,
	})

	kctx, err := mock.BuildContext(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if kctx.StratumCeiling != 850_000 {
		t.Fatalf("expected 850000, got %d", kctx.StratumCeiling)
	}

	_, err = mock.BuildContext(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing claim")
	}
}
