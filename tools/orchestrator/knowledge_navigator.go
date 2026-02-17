package orchestrator

import (
	"context"
	"fmt"
	"sort"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ReferenceSubgraph is the result of a BFS traversal from a fact through references.
type ReferenceSubgraph struct {
	Nodes []agentsdk.FactInfo
	Edges []agentsdk.FactReference
	Depth int
}

// CitationLevel holds facts at a specific depth in a citation chain.
type CitationLevel struct {
	Level int
	Facts []agentsdk.FactInfo
}

// CitationChain traces supporting references from a fact to its foundations.
type CitationChain struct {
	FactID     string
	Ancestors  []CitationLevel // level 0 = direct refs, level 1 = their refs, etc.
	MaxDepth   int
	Foundation []string // leaf fact IDs with no outgoing supporting refs
}

// KnowledgeNavigator traverses the knowledge graph and analyzes reference structures.
type KnowledgeNavigator interface {
	// FollowReferences does BFS from a fact up to maxDepth, returning the subgraph.
	FollowReferences(ctx context.Context, factID string, maxDepth int) (*ReferenceSubgraph, error)

	// ScoreNovelty computes how novel a claim is relative to existing domain facts.
	// Returns 0-10000 BPS (10000 = completely novel).
	ScoreNovelty(ctx context.Context, claim *agentsdk.ClaimInfo, domainFacts []agentsdk.FactInfo) uint32

	// FindCitationChain finds the chain of supporting refs from a fact to its foundations.
	FindCitationChain(ctx context.Context, factID string) (*CitationChain, error)
}

// KnowledgeGap represents an area where the knowledge graph is weak or incomplete.
type KnowledgeGap struct {
	Type         string   `json:"type"`          // "low_density", "decaying", "unchallenged", "cross_domain"
	Domain       string   `json:"domain"`
	Description  string   `json:"description"`
	Score        uint32   `json:"score"`         // priority score (BPS), higher = more urgent
	RelatedFacts []string `json:"related_facts"`
}

// GapDetector identifies weaknesses in the knowledge graph.
type GapDetector interface {
	DetectGaps(ctx context.Context, domain string) ([]KnowledgeGap, error)
}

// DefaultKnowledgeNavigator implements KnowledgeNavigator using a QueryClient.
type DefaultKnowledgeNavigator struct {
	qc agentsdk.QueryClient
}

// NewDefaultKnowledgeNavigator creates a navigator backed by a QueryClient.
func NewDefaultKnowledgeNavigator(qc agentsdk.QueryClient) *DefaultKnowledgeNavigator {
	return &DefaultKnowledgeNavigator{qc: qc}
}

// FollowReferences does BFS from a fact up to maxDepth.
func (n *DefaultKnowledgeNavigator) FollowReferences(ctx context.Context, factID string, maxDepth int) (*ReferenceSubgraph, error) {
	if n.qc == nil {
		return nil, fmt.Errorf("query client not set")
	}
	if maxDepth <= 0 {
		maxDepth = 1
	}

	subgraph := &ReferenceSubgraph{Depth: maxDepth}
	visited := make(map[string]bool)
	frontier := []string{factID}

	for depth := 0; depth < maxDepth && len(frontier) > 0; depth++ {
		var nextFrontier []string

		for _, fid := range frontier {
			if visited[fid] {
				continue
			}
			visited[fid] = true

			refs, err := n.qc.FactReferences(ctx, fid)
			if err != nil {
				continue // graceful degradation
			}

			for _, ref := range refs {
				subgraph.Edges = append(subgraph.Edges, ref)
				if !visited[ref.ToFactID] {
					nextFrontier = append(nextFrontier, ref.ToFactID)
				}
			}
		}

		frontier = nextFrontier
	}

	// Resolve node info for all visited facts by querying the domain
	// In practice this would be a batch query; for now we record IDs as minimal nodes.
	for fid := range visited {
		subgraph.Nodes = append(subgraph.Nodes, agentsdk.FactInfo{ID: fid})
	}

	// Sort nodes by ID for determinism
	sort.Slice(subgraph.Nodes, func(i, j int) bool {
		return subgraph.Nodes[i].ID < subgraph.Nodes[j].ID
	})

	// Sort edges by FromFactID, then ToFactID for determinism
	sort.Slice(subgraph.Edges, func(i, j int) bool {
		if subgraph.Edges[i].FromFactID != subgraph.Edges[j].FromFactID {
			return subgraph.Edges[i].FromFactID < subgraph.Edges[j].FromFactID
		}
		return subgraph.Edges[i].ToFactID < subgraph.Edges[j].ToFactID
	})

	return subgraph, nil
}

// ScoreNovelty computes how novel a claim is relative to existing domain facts.
// Uses text comparison heuristic: if any existing fact has the same text, novelty is 0.
// Otherwise novelty increases with domain density (more facts = harder to be novel).
func (n *DefaultKnowledgeNavigator) ScoreNovelty(_ context.Context, claim *agentsdk.ClaimInfo, domainFacts []agentsdk.FactInfo) uint32 {
	if claim == nil {
		return 0
	}
	if len(domainFacts) == 0 {
		return 10000 // completely novel in empty domain
	}

	// Check for exact text match
	for _, f := range domainFacts {
		if f.ClaimText == claim.Text {
			return 0 // duplicate
		}
	}

	// Novelty inversely proportional to domain density, capped at 10000
	// More facts in domain → harder to claim novelty (but never 0 for non-duplicates)
	// Formula: 10000 - min(densityPenalty, 8000)
	// densityPenalty = factCount * 100 (each fact reduces novelty by 1%)
	densityPenalty := uint32(len(domainFacts)) * 100
	if densityPenalty > 8000 {
		densityPenalty = 8000
	}

	return 10000 - densityPenalty
}

// FindCitationChain traces supporting references from a fact to its leaf foundations.
func (n *DefaultKnowledgeNavigator) FindCitationChain(ctx context.Context, factID string) (*CitationChain, error) {
	if n.qc == nil {
		return nil, fmt.Errorf("query client not set")
	}

	chain := &CitationChain{
		FactID: factID,
	}

	visited := make(map[string]bool)
	frontier := []string{factID}
	visited[factID] = true
	level := 0

	for len(frontier) > 0 && level < 10 { // max depth 10 to prevent infinite loops
		var levelFacts []agentsdk.FactInfo
		var nextFrontier []string

		for _, fid := range frontier {
			refs, err := n.qc.FactReferences(ctx, fid)
			if err != nil {
				continue
			}

			hasSupportingRef := false
			for _, ref := range refs {
				if ref.Type == "supports" && !visited[ref.ToFactID] {
					hasSupportingRef = true
					visited[ref.ToFactID] = true
					nextFrontier = append(nextFrontier, ref.ToFactID)
					levelFacts = append(levelFacts, agentsdk.FactInfo{ID: ref.ToFactID})
				}
			}

			// If no supporting refs, this is a foundation node
			if !hasSupportingRef && fid != factID {
				chain.Foundation = append(chain.Foundation, fid)
			}
		}

		if len(levelFacts) > 0 {
			// Sort level facts by ID for determinism
			sort.Slice(levelFacts, func(i, j int) bool {
				return levelFacts[i].ID < levelFacts[j].ID
			})
			chain.Ancestors = append(chain.Ancestors, CitationLevel{
				Level: level,
				Facts: levelFacts,
			})
		}

		frontier = nextFrontier
		level++
	}

	chain.MaxDepth = level

	// If any remaining frontier nodes have no further refs, add as foundation
	if len(frontier) > 0 {
		for _, fid := range frontier {
			chain.Foundation = append(chain.Foundation, fid)
		}
	}

	// Sort foundation for determinism
	sort.Strings(chain.Foundation)

	return chain, nil
}

// DefaultGapDetector identifies knowledge gaps in a domain.
type DefaultGapDetector struct {
	qc             agentsdk.QueryClient
	minDensity     uint32 // domains below this fact count are "low density"
	decayThreshold uint32 // confidence below this flags decaying facts
}

// NewDefaultGapDetector creates a gap detector with standard thresholds.
func NewDefaultGapDetector(qc agentsdk.QueryClient) *DefaultGapDetector {
	return &DefaultGapDetector{
		qc:             qc,
		minDensity:     10,
		decayThreshold: 400_000,
	}
}

// DetectGaps scans a domain for knowledge weaknesses.
func (d *DefaultGapDetector) DetectGaps(ctx context.Context, domain string) ([]KnowledgeGap, error) {
	if d.qc == nil {
		return nil, fmt.Errorf("query client not set")
	}

	facts, err := d.qc.FactsByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("querying facts for domain %s: %w", domain, err)
	}

	var gaps []KnowledgeGap

	// Gap type 1: Low density
	if uint32(len(facts)) < d.minDensity {
		gaps = append(gaps, KnowledgeGap{
			Type:        "low_density",
			Domain:      domain,
			Description: fmt.Sprintf("domain has only %d facts (threshold: %d)", len(facts), d.minDensity),
			Score:       (d.minDensity - uint32(len(facts))) * 1000, // 1000 BPS per missing fact
		})
	}

	// Gap type 2: Decaying facts (confidence below threshold)
	var decayingIDs []string
	for _, f := range facts {
		if f.Confidence < d.decayThreshold && f.Confidence > 0 {
			decayingIDs = append(decayingIDs, f.ID)
		}
	}
	if len(decayingIDs) > 0 {
		sort.Strings(decayingIDs) // deterministic
		score := uint32(len(decayingIDs)) * 500
		if score > 10000 {
			score = 10000
		}
		gaps = append(gaps, KnowledgeGap{
			Type:         "decaying",
			Domain:       domain,
			Description:  fmt.Sprintf("%d facts below confidence threshold %d", len(decayingIDs), d.decayThreshold),
			Score:        score,
			RelatedFacts: decayingIDs,
		})
	}

	// Gap type 3: Unchallenged high-confidence facts
	var unchallengedIDs []string
	for _, f := range facts {
		if f.Confidence >= 800_000 {
			// Check for counter-facts
			counterFacts, err := d.qc.CounterFacts(ctx, f.ID)
			if err != nil {
				continue
			}
			if len(counterFacts) == 0 {
				unchallengedIDs = append(unchallengedIDs, f.ID)
			}
		}
	}
	if len(unchallengedIDs) > 0 {
		sort.Strings(unchallengedIDs) // deterministic
		score := uint32(len(unchallengedIDs)) * 300
		if score > 10000 {
			score = 10000
		}
		gaps = append(gaps, KnowledgeGap{
			Type:         "unchallenged",
			Domain:       domain,
			Description:  fmt.Sprintf("%d high-confidence facts never challenged", len(unchallengedIDs)),
			Score:        score,
			RelatedFacts: unchallengedIDs,
		})
	}

	// Gap type 4: Cross-domain references (facts referencing other domains)
	var crossDomainIDs []string
	for _, f := range facts {
		for _, refID := range f.References {
			refs, err := d.qc.FactReferences(ctx, refID)
			if err != nil {
				continue
			}
			for _, ref := range refs {
				// We can't easily check the domain of the target fact without
				// fetching it, so we flag any fact with outgoing references as
				// a potential cross-domain bridge.
				if ref.Type == "extends" {
					crossDomainIDs = append(crossDomainIDs, f.ID)
					break
				}
			}
		}
	}
	if len(crossDomainIDs) > 0 {
		sort.Strings(crossDomainIDs) // deterministic
		// Deduplicate
		crossDomainIDs = dedup(crossDomainIDs)
		score := uint32(len(crossDomainIDs)) * 200
		if score > 10000 {
			score = 10000
		}
		gaps = append(gaps, KnowledgeGap{
			Type:         "cross_domain",
			Domain:       domain,
			Description:  fmt.Sprintf("%d facts with cross-domain extension references", len(crossDomainIDs)),
			Score:        score,
			RelatedFacts: crossDomainIDs,
		})
	}

	// Sort gaps by score desc, type asc for tiebreaker
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Score != gaps[j].Score {
			return gaps[i].Score > gaps[j].Score
		}
		return gaps[i].Type < gaps[j].Type
	})

	return gaps, nil
}

// dedup removes duplicate strings from a sorted slice.
func dedup(sorted []string) []string {
	if len(sorted) <= 1 {
		return sorted
	}
	result := []string{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			result = append(result, sorted[i])
		}
	}
	return result
}

// MockKnowledgeNavigator is a test-only navigator.
type MockKnowledgeNavigator struct {
	Subgraphs  map[string]*ReferenceSubgraph  // factID → subgraph
	Chains     map[string]*CitationChain       // factID → chain
	NoveltyVal uint32
}

// NewMockKnowledgeNavigator creates a mock navigator.
func NewMockKnowledgeNavigator() *MockKnowledgeNavigator {
	return &MockKnowledgeNavigator{
		Subgraphs:  make(map[string]*ReferenceSubgraph),
		Chains:     make(map[string]*CitationChain),
		NoveltyVal: 5000,
	}
}

func (m *MockKnowledgeNavigator) FollowReferences(_ context.Context, factID string, _ int) (*ReferenceSubgraph, error) {
	if sg, ok := m.Subgraphs[factID]; ok {
		return sg, nil
	}
	return &ReferenceSubgraph{}, nil
}

func (m *MockKnowledgeNavigator) ScoreNovelty(_ context.Context, _ *agentsdk.ClaimInfo, _ []agentsdk.FactInfo) uint32 {
	return m.NoveltyVal
}

func (m *MockKnowledgeNavigator) FindCitationChain(_ context.Context, factID string) (*CitationChain, error) {
	if ch, ok := m.Chains[factID]; ok {
		return ch, nil
	}
	return &CitationChain{FactID: factID}, nil
}

// MockGapDetector is a test-only gap detector.
type MockGapDetector struct {
	Gaps map[string][]KnowledgeGap // domain → gaps
}

// NewMockGapDetector creates a mock gap detector.
func NewMockGapDetector() *MockGapDetector {
	return &MockGapDetector{
		Gaps: make(map[string][]KnowledgeGap),
	}
}

func (m *MockGapDetector) DetectGaps(_ context.Context, domain string) ([]KnowledgeGap, error) {
	if g, ok := m.Gaps[domain]; ok {
		return g, nil
	}
	return nil, nil
}

// Compile-time checks
var (
	_ KnowledgeNavigator = (*DefaultKnowledgeNavigator)(nil)
	_ KnowledgeNavigator = (*MockKnowledgeNavigator)(nil)
	_ GapDetector        = (*DefaultGapDetector)(nil)
	_ GapDetector        = (*MockGapDetector)(nil)
)
