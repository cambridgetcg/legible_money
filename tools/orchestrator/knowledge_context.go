package orchestrator

import (
	"context"
	"fmt"
	"sort"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// DomainContext holds aggregated knowledge about a domain.
type DomainContext struct {
	Name        string
	FactCount   uint32
	TopFacts    []agentsdk.FactInfo   // highest confidence facts, sorted desc with ID tiebreaker
	Categories  map[string]uint32     // category → count in domain
	MeanConfBps uint32                // mean confidence in BPS (0 if empty)
}

// KnowledgeContext is the rich evaluation context assembled from chain state.
type KnowledgeContext struct {
	Claim          *agentsdk.ClaimInfo
	Domain         DomainContext
	SupportingFacts []agentsdk.FactInfo      // facts that support the claim's references
	CounterFacts   []agentsdk.CounterFactInfo // counter-facts targeting the claim
	ReferenceChain []agentsdk.FactReference   // reference edges from cited facts
	StratumCeiling uint32                     // max confidence from ontology stratum
	LogicZoneCap   uint32                     // max confidence from logic zone
	DomainDensity  uint32                     // number of facts in domain
}

// KnowledgeContextProvider builds a KnowledgeContext for claim evaluation.
type KnowledgeContextProvider interface {
	BuildContext(ctx context.Context, claimID string) (*KnowledgeContext, error)
}

// Ontological stratum ceilings (BPS, duplicated from x/knowledge for off-chain use).
var stratumCeilings = map[string]uint32{
	"fundamental":   1_000_000,
	"physical":      990_000,
	"chemical":      980_000,
	"biological":    950_000,
	"cognitive":     900_000,
	"social":        850_000,
	"technological": 1_000_000,
}

// Logic zone confidence caps (BPS).
var logicZoneCaps = map[string]uint32{
	"propositional": 1_000_000,
	"presburger":    1_000_000,
	"peano":         850_000,
	"zfc":           800_000,
	"empirical":     700_000,
}

const (
	defaultStratumCeiling = 900_000
	defaultLogicZoneCap   = 700_000
	maxTopFacts           = 10
)

// LookupStratumCeiling returns the stratum ceiling for a domain name.
// Falls back to defaultStratumCeiling for unknown domains.
func LookupStratumCeiling(domain string) uint32 {
	if ceil, ok := stratumCeilings[domain]; ok {
		return ceil
	}
	return defaultStratumCeiling
}

// LookupLogicZoneCap returns the logic zone cap for a zone name.
// Falls back to defaultLogicZoneCap for unknown zones.
func LookupLogicZoneCap(zone string) uint32 {
	if cap, ok := logicZoneCaps[zone]; ok {
		return cap
	}
	return defaultLogicZoneCap
}

// QueryClientContextProvider implements KnowledgeContextProvider using a QueryClient.
type QueryClientContextProvider struct {
	qc agentsdk.QueryClient
}

// NewQueryClientContextProvider creates a context provider backed by a QueryClient.
func NewQueryClientContextProvider(qc agentsdk.QueryClient) *QueryClientContextProvider {
	return &QueryClientContextProvider{qc: qc}
}

// BuildContext assembles a KnowledgeContext for the given claim.
// Gracefully degrades when individual queries fail (returns zero-valued fields).
func (p *QueryClientContextProvider) BuildContext(ctx context.Context, claimID string) (*KnowledgeContext, error) {
	if p.qc == nil {
		return nil, fmt.Errorf("query client not set")
	}

	// 1. Get claim info
	claim, err := p.qc.GetClaim(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("getting claim %s: %w", claimID, err)
	}

	kctx := &KnowledgeContext{
		Claim: claim,
	}

	// 2. Get domain facts and build domain context
	domainFacts, err := p.qc.FactsByDomain(ctx, claim.Domain)
	if err != nil {
		// Graceful degradation: empty domain
		domainFacts = nil
	}
	kctx.Domain = buildDomainContext(claim.Domain, domainFacts)
	kctx.DomainDensity = kctx.Domain.FactCount

	// 3. For each fact in domain that references other facts, gather reference chains
	for _, fact := range domainFacts {
		for _, refID := range fact.References {
			refs, err := p.qc.FactReferences(ctx, refID)
			if err != nil {
				continue // graceful degradation
			}
			kctx.ReferenceChain = append(kctx.ReferenceChain, refs...)
			// Track supporting facts
			for _, ref := range refs {
				if ref.Type == "supports" {
					// Look up the supporting fact in domain facts
					for _, df := range domainFacts {
						if df.ID == ref.ToFactID {
							kctx.SupportingFacts = append(kctx.SupportingFacts, df)
						}
					}
				}
			}
		}
	}

	// 4. Get counter-facts for the claim
	counterFacts, err := p.qc.CounterFacts(ctx, claimID)
	if err != nil {
		counterFacts = nil // graceful degradation
	}
	kctx.CounterFacts = counterFacts

	// 5. Look up stratum ceiling and logic zone cap
	kctx.StratumCeiling = LookupStratumCeiling(claim.Domain)
	kctx.LogicZoneCap = defaultLogicZoneCap // Default; overridden if facts have LogicZone set

	// If we have domain facts with logic zones, use the most common one
	if len(domainFacts) > 0 {
		zoneCounts := make(map[string]int)
		for _, f := range domainFacts {
			if f.LogicZone != "" {
				zoneCounts[f.LogicZone]++
			}
		}
		bestZone := ""
		bestCount := 0
		for zone, count := range zoneCounts {
			if count > bestCount || (count == bestCount && zone < bestZone) {
				bestZone = zone
				bestCount = count
			}
		}
		if bestZone != "" {
			kctx.LogicZoneCap = LookupLogicZoneCap(bestZone)
		}
	}

	return kctx, nil
}

// buildDomainContext aggregates facts into a DomainContext.
func buildDomainContext(domain string, facts []agentsdk.FactInfo) DomainContext {
	dc := DomainContext{
		Name:       domain,
		FactCount:  uint32(len(facts)),
		Categories: make(map[string]uint32),
	}

	if len(facts) == 0 {
		return dc
	}

	// Aggregate categories and compute mean confidence
	var totalConf uint64
	for _, f := range facts {
		dc.Categories[f.Category]++
		totalConf += uint64(f.Confidence)
	}
	dc.MeanConfBps = uint32(totalConf / uint64(len(facts)))

	// Sort by confidence desc, ID asc for tiebreaker, take top N
	sorted := make([]agentsdk.FactInfo, len(facts))
	copy(sorted, facts)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Confidence != sorted[j].Confidence {
			return sorted[i].Confidence > sorted[j].Confidence
		}
		return sorted[i].ID < sorted[j].ID
	})

	topN := maxTopFacts
	if len(sorted) < topN {
		topN = len(sorted)
	}
	dc.TopFacts = sorted[:topN]

	return dc
}

// MockKnowledgeContextProvider is a test-only provider with configurable responses.
type MockKnowledgeContextProvider struct {
	Contexts map[string]*KnowledgeContext // claimID → context
}

// NewMockKnowledgeContextProvider creates a mock provider.
func NewMockKnowledgeContextProvider() *MockKnowledgeContextProvider {
	return &MockKnowledgeContextProvider{
		Contexts: make(map[string]*KnowledgeContext),
	}
}

// AddContext adds a context for a claim.
func (m *MockKnowledgeContextProvider) AddContext(claimID string, kctx *KnowledgeContext) {
	m.Contexts[claimID] = kctx
}

// BuildContext returns the mock context or an error.
func (m *MockKnowledgeContextProvider) BuildContext(_ context.Context, claimID string) (*KnowledgeContext, error) {
	if kctx, ok := m.Contexts[claimID]; ok {
		return kctx, nil
	}
	return nil, fmt.Errorf("mock context not found: %s", claimID)
}

// Compile-time checks
var (
	_ KnowledgeContextProvider = (*QueryClientContextProvider)(nil)
	_ KnowledgeContextProvider = (*MockKnowledgeContextProvider)(nil)
)
