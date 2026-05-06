package keeper

import (
	"encoding/json"
	"fmt"
	"sort"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- FARM-11: Per-Submitter Citation Concentration Detection ----------
//
// The existing per-fact CitationSourceRecord (citation.go) detects when a single
// fact is being cited by a sybil ring. This file adds the complementary view:
// detecting when a *submitter* concentrates their outgoing citations toward a
// small set of other submitters (cartel pattern).
//
// Key: SubmitterCitationGraphPrefix/{submitter} -> SubmitterCitationGraph
// Updated on every claim submission that includes references.
// Queried during reward distribution.

// submitterCitationGraphKey returns the KV key for a submitter's citation graph.
func submitterCitationGraphKey(submitter string) []byte {
	return append(types.SubmitterCitationGraphPrefix, []byte(submitter)...)
}

// GetSubmitterCitationGraph retrieves the citation graph for a submitter.
func (k Keeper) GetSubmitterCitationGraph(ctx sdk.Context, submitter string) (types.SubmitterCitationGraph, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(submitterCitationGraphKey(submitter))
	if err != nil || bz == nil {
		return types.SubmitterCitationGraph{}, false
	}
	var graph types.SubmitterCitationGraph
	if err := json.Unmarshal(bz, &graph); err != nil {
		return types.SubmitterCitationGraph{}, false
	}
	return graph, true
}

// SetSubmitterCitationGraph stores the citation graph for a submitter.
func (k Keeper) SetSubmitterCitationGraph(ctx sdk.Context, graph types.SubmitterCitationGraph) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(graph)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal submitter citation graph: %v", err))
	}
	_ = kvStore.Set(submitterCitationGraphKey(graph.Submitter), bz)
}

// UpdateSubmitterCitationGraph records that submitter cited a fact owned by citedSubmitter.
// Called during claim submission when references are processed.
func (k Keeper) UpdateSubmitterCitationGraph(ctx sdk.Context, submitter, citedSubmitter string) {
	if submitter == "" || citedSubmitter == "" {
		return
	}
	graph, found := k.GetSubmitterCitationGraph(ctx, submitter)
	if !found {
		graph = types.SubmitterCitationGraph{
			Submitter:   submitter,
			CitedCounts: make(map[string]uint64),
		}
	}
	graph.CitedCounts[citedSubmitter]++
	graph.TotalCites++
	k.SetSubmitterCitationGraph(ctx, graph)
}

// CheckCitationConcentration computes the concentration score for a submitter's
// outgoing citations. Returns the concentration score (0-1_000_000 scale, where
// 1_000_000 = 100% concentrated) and the list of top-cited submitter addresses.
//
// Concentration is measured as the share of total citations that go to the top N
// most-cited submitters, where N = CitationConcentrationMaxAddrs (default 3).
//
// A score > CitationConcentrationThreshold (default 800_000 = 80%) means the
// submitter is concentrating citations among a cartel-sized group.
func (k Keeper) CheckCitationConcentration(
	ctx sdk.Context,
	submitter string,
	refs []string,
) (concentrationScore uint64, topCited []string) {
	graph, found := k.GetSubmitterCitationGraph(ctx, submitter)
	if !found || graph.TotalCites == 0 {
		return 0, nil
	}

	ep := k.GetExtendedParams(ctx)
	if ep.CitationConcentrationMaxAddrs == 0 {
		return 0, nil
	}

	// Not enough data to judge
	if graph.TotalCites < ep.CitationConcentrationMinCites {
		return 0, nil
	}

	// Sort submitters by citation count descending
	type addrCount struct {
		addr  string
		count uint64
	}
	sorted := make([]addrCount, 0, len(graph.CitedCounts))
	for addr, count := range graph.CitedCounts {
		sorted = append(sorted, addrCount{addr, count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

	// Sum top N
	var topSum uint64
	n := ep.CitationConcentrationMaxAddrs
	for i := uint64(0); i < n && i < uint64(len(sorted)); i++ {
		topSum += sorted[i].count
		topCited = append(topCited, sorted[i].addr)
	}

	// concentration = topSum * 1_000_000 / totalCites
	concentrationScore = (topSum * 1_000_000) / graph.TotalCites
	return concentrationScore, topCited
}

// ApplyCitationConcentrationPenalty reduces a base citation reward proportionally
// to the submitter's citation concentration score.
//
// If the submitter's outgoing citations are concentrated above the threshold
// (default 80%), the reward is reduced by the concentration percentage.
// Example: 90% concentration -> reward reduced by 90%.
//
// This is the per-submitter penalty (complements the per-fact discount in
// GetCitationConcentrationDiscount).
func (k Keeper) ApplyCitationConcentrationPenalty(
	ctx sdk.Context,
	submitter string,
	baseReward sdkmath.Int,
) sdkmath.Int {
	if baseReward.IsZero() || baseReward.IsNegative() {
		return baseReward
	}

	ep := k.GetExtendedParams(ctx)
	if ep.CitationConcentrationThreshold == 0 || ep.CitationConcentrationDiscount == 0 {
		return baseReward
	}

	graph, found := k.GetSubmitterCitationGraph(ctx, submitter)
	if !found || graph.TotalCites < ep.CitationConcentrationMinCites {
		return baseReward
	}

	topN := TopNSum(graph.CitedCounts, ep.CitationConcentrationMaxAddrs)

	// concentration = topN * 1_000_000 / totalCites
	concentration := (topN * 1_000_000) / graph.TotalCites

	if concentration < ep.CitationConcentrationThreshold {
		return baseReward
	}

	// Reduce reward proportionally to concentration.
	// penalty = baseReward * concentration / 1_000_000
	penalty := baseReward.Mul(sdkmath.NewIntFromUint64(concentration)).Quo(sdkmath.NewIntFromUint64(1_000_000))
	result := baseReward.Sub(penalty)
	if result.IsNegative() {
		return sdkmath.ZeroInt()
	}
	return result
}
