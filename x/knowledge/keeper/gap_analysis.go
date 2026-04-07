package keeper

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// GapReport summarizes knowledge graph health and growth opportunities.
type GapReport struct {
	TotalFacts    int            `json:"total_facts"`
	FactsByDomain map[string]int `json:"facts_by_domain"`
	FactsByStatus map[string]int `json:"facts_by_status"`

	// Gaps identified
	UncitedAxioms  []GapEntry `json:"uncited_axioms"`  // High-confidence facts with 0 incoming citations
	SparseDomains  []GapEntry `json:"sparse_domains"`  // Domains with < N facts
	CitationOrphans []GapEntry `json:"citation_orphans"` // Facts referencing non-existent or revoked facts
	ShallowTrees   []GapEntry `json:"shallow_trees"`   // Facts where all citers are same submitter (FARM-11)
	StaleFacts     []GapEntry `json:"stale_facts"`     // Facts with no confidence growth in N epochs

	// Priorities (sorted by impact score, highest first)
	TopGaps []GapEntry `json:"top_gaps"`
}

// GapEntry describes a single identified gap in the knowledge graph.
type GapEntry struct {
	FactID     string `json:"fact_id"`
	Domain     string `json:"domain"`
	Type       string `json:"type"`       // "uncited_axiom", "sparse_domain", "citation_orphan", "shallow_tree", "stale"
	Priority   uint64 `json:"priority"`   // Higher = more important to fill
	Suggestion string `json:"suggestion"` // Human-readable suggestion for what to submit
}

// Gap analysis thresholds — tunable per deployment.
const (
	// Confidence threshold for an axiom to be considered "high confidence" (basis points).
	// 80% = 800_000 bps.
	AxiomConfidenceThreshold uint64 = 800_000

	// Minimum facts in a domain before it is considered sparse.
	SparseDomainThreshold = 5

	// Number of epochs (blocks) without confidence growth before a fact is stale.
	// ~7 days at 34,272 blocks/day ≈ 240,000 blocks.
	StaleEpochThreshold uint64 = 240_000

	// Priority weights for gap types (used to rank TopGaps).
	PriorityUncitedAxiom  uint64 = 1000
	PrioritySparseDomain  uint64 = 800
	PriorityCitationOrphan uint64 = 600
	PriorityShallowTree   uint64 = 500
	PriorityStale         uint64 = 400
)

// AnalyzeGaps iterates the knowledge graph and builds a comprehensive gap report.
func (k Keeper) AnalyzeGaps(ctx sdk.Context) GapReport {
	report := GapReport{
		FactsByDomain: make(map[string]int),
		FactsByStatus: make(map[string]int),
	}

	// Collect all fact IDs in a set for orphan detection.
	factIDs := make(map[string]bool)
	// Collect all facts for multi-pass analysis.
	var allFacts []*types.Fact

	currentHeight := uint64(ctx.BlockHeight())

	// Single iteration: collect counts and the full fact list.
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		report.TotalFacts++
		report.FactsByDomain[f.Domain]++
		report.FactsByStatus[f.Status]++
		factIDs[f.Id] = true
		allFacts = append(allFacts, f)
		return false
	})

	// Pass 1: Uncited axioms, citation orphans, shallow trees, stale facts.
	for _, f := range allFacts {
		// --- Uncited axioms ---
		if isActiveOrVerified(f.Status) && f.Confidence >= AxiomConfidenceThreshold {
			incoming := k.GetIncomingReferences(ctx, f.Id)
			if len(incoming) == 0 {
				entry := GapEntry{
					FactID:   f.Id,
					Domain:   f.Domain,
					Type:     "uncited_axiom",
					Priority: PriorityUncitedAxiom + f.Confidence/1000, // Higher confidence = higher priority
				}
				entry.Suggestion = GenerateSuggestion(entry)
				report.UncitedAxioms = append(report.UncitedAxioms, entry)
			}
		}

		// --- Citation orphans ---
		refs := k.GetFactReferences(ctx, f.Id)
		for _, ref := range refs {
			targetFact, found := k.GetFact(ctx, ref.ToFactId)
			if !found || targetFact.Status == types.FactStatusRevoked {
				entry := GapEntry{
					FactID:   f.Id,
					Domain:   f.Domain,
					Type:     "citation_orphan",
					Priority: PriorityCitationOrphan + f.Confidence/1000,
				}
				entry.Suggestion = GenerateSuggestion(entry)
				report.CitationOrphans = append(report.CitationOrphans, entry)
				break // One orphan ref is enough to flag the fact
			}
		}

		// --- Shallow trees (FARM-11) ---
		if f.CitationCount > 0 {
			record, found := k.GetCitationSourceRecord(ctx, f.Id)
			if found && record.TotalCites > 0 && len(record.SourceCounts) == 1 {
				entry := GapEntry{
					FactID:   f.Id,
					Domain:   f.Domain,
					Type:     "shallow_tree",
					Priority: PriorityShallowTree + f.CitationCount*10,
				}
				entry.Suggestion = GenerateSuggestion(entry)
				report.ShallowTrees = append(report.ShallowTrees, entry)
			}
		}

		// --- Stale facts ---
		if isActiveOrVerified(f.Status) && currentHeight > 0 {
			lastUpdate := f.LastConfidenceUpdate
			if lastUpdate == 0 {
				lastUpdate = f.VerifiedAtHeight
			}
			if lastUpdate > 0 && currentHeight-lastUpdate > StaleEpochThreshold {
				entry := GapEntry{
					FactID:   f.Id,
					Domain:   f.Domain,
					Type:     "stale",
					Priority: PriorityStale + (currentHeight-lastUpdate)/1000,
				}
				entry.Suggestion = GenerateSuggestion(entry)
				report.StaleFacts = append(report.StaleFacts, entry)
			}
		}
	}

	// Pass 2: Sparse domains — check registered domains that have few facts.
	k.IterateDomains(ctx, func(d *types.OntologicalDomain) bool {
		count := report.FactsByDomain[d.Name]
		if count < SparseDomainThreshold {
			entry := GapEntry{
				Domain:   d.Name,
				Type:     "sparse_domain",
				Priority: PrioritySparseDomain + uint64(SparseDomainThreshold-count)*100,
			}
			entry.Suggestion = GenerateSuggestion(entry)
			report.SparseDomains = append(report.SparseDomains, entry)
		}
		return false
	})

	// Build TopGaps: merge all gap entries and sort by priority descending.
	report.TopGaps = mergeAndSortGaps(
		report.UncitedAxioms,
		report.SparseDomains,
		report.CitationOrphans,
		report.ShallowTrees,
		report.StaleFacts,
	)

	return report
}

// GetUncitedAxioms returns facts with status "verified"/"active" and confidence > 80%
// but zero incoming citations. These are genesis axioms that haven't sprouted.
func (k Keeper) GetUncitedAxioms(ctx sdk.Context) []GapEntry {
	var entries []GapEntry
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if !isActiveOrVerified(f.Status) || f.Confidence < AxiomConfidenceThreshold {
			return false
		}
		incoming := k.GetIncomingReferences(ctx, f.Id)
		if len(incoming) == 0 {
			entry := GapEntry{
				FactID:   f.Id,
				Domain:   f.Domain,
				Type:     "uncited_axiom",
				Priority: PriorityUncitedAxiom + f.Confidence/1000,
			}
			entry.Suggestion = GenerateSuggestion(entry)
			entries = append(entries, entry)
		}
		return false
	})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority > entries[j].Priority
	})
	return entries
}

// GetSparseDomains returns domains with fewer than SparseDomainThreshold facts.
func (k Keeper) GetSparseDomains(ctx sdk.Context) []GapEntry {
	// Count facts per domain.
	domainCounts := make(map[string]int)
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		domainCounts[f.Domain]++
		return false
	})

	var entries []GapEntry
	k.IterateDomains(ctx, func(d *types.OntologicalDomain) bool {
		count := domainCounts[d.Name]
		if count < SparseDomainThreshold {
			entry := GapEntry{
				Domain:   d.Name,
				Type:     "sparse_domain",
				Priority: PrioritySparseDomain + uint64(SparseDomainThreshold-count)*100,
			}
			entry.Suggestion = GenerateSuggestion(entry)
			entries = append(entries, entry)
		}
		return false
	})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority > entries[j].Priority
	})
	return entries
}

// GetCitationOrphans returns facts whose outgoing references point to non-existent
// or revoked facts.
func (k Keeper) GetCitationOrphans(ctx sdk.Context) []GapEntry {
	var entries []GapEntry
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		refs := k.GetFactReferences(ctx, f.Id)
		for _, ref := range refs {
			target, found := k.GetFact(ctx, ref.ToFactId)
			if !found || target.Status == types.FactStatusRevoked {
				entry := GapEntry{
					FactID:   f.Id,
					Domain:   f.Domain,
					Type:     "citation_orphan",
					Priority: PriorityCitationOrphan + f.Confidence/1000,
				}
				entry.Suggestion = GenerateSuggestion(entry)
				entries = append(entries, entry)
				break
			}
		}
		return false
	})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority > entries[j].Priority
	})
	return entries
}

// GenerateSuggestion creates a human-readable suggestion for filling a knowledge gap.
func GenerateSuggestion(gap GapEntry) string {
	switch gap.Type {
	case "uncited_axiom":
		return fmt.Sprintf(
			"Fact %s in domain %q has high confidence but no citations. Submit claims that reference this axiom to grow the knowledge tree.",
			gap.FactID, gap.Domain,
		)
	case "sparse_domain":
		return fmt.Sprintf(
			"Domain %q has fewer than %d facts. Submit new claims in this domain to improve coverage.",
			gap.Domain, SparseDomainThreshold,
		)
	case "citation_orphan":
		return fmt.Sprintf(
			"Fact %s in domain %q references a non-existent or revoked fact. Submit a replacement reference or update the claim.",
			gap.FactID, gap.Domain,
		)
	case "shallow_tree":
		return fmt.Sprintf(
			"Fact %s in domain %q is only cited by a single submitter. Encourage independent citations to strengthen the knowledge graph.",
			gap.FactID, gap.Domain,
		)
	case "stale":
		return fmt.Sprintf(
			"Fact %s in domain %q has not grown in confidence for a long time. Consider re-verification or submitting supporting evidence.",
			gap.FactID, gap.Domain,
		)
	default:
		return "Unknown gap type."
	}
}

// isActiveOrVerified returns true if status is "verified" or "active".
func isActiveOrVerified(status string) bool {
	return status == types.FactStatusVerified || status == types.FactStatusActive
}

// mergeAndSortGaps combines multiple gap slices and sorts by priority descending.
func mergeAndSortGaps(lists ...[]GapEntry) []GapEntry {
	var all []GapEntry
	for _, list := range lists {
		all = append(all, list...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Priority > all[j].Priority
	})
	// Cap at 100 entries to keep the report manageable.
	if len(all) > 100 {
		all = all[:100]
	}
	return all
}
