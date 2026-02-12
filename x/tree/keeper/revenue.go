package keeper

import (
	"sort"

	"github.com/legible-money/legible/x/tree/types"
)

// Revenue basis point constants (1000000 = 100%)
const (
	BpDenominator = int64(1000000)

	// Evidence tax 5-way sub-split (out of 1000000)
	TaxResearchBp   = int64(770000)
	TaxFoundationBp = int64(70000)
	TaxCommunityBp  = int64(70000)
	TaxDevelopersBp = int64(70000)
	TaxReserveBp    = int64(20000)
)

// RevenueDistribution holds the computed split for a service call payment.
type RevenueDistribution struct {
	EvidenceTax      int64
	ContributorPool  int64
	ResearchFund     int64
	ProtocolTreasury int64

	// Evidence tax 5-way sub-split
	TaxResearch   int64
	TaxFoundation int64
	TaxCommunity  int64
	TaxDevelopers int64
	TaxReserve    int64

	// Per-contributor shares (sorted by address for determinism)
	ContributorShares []ContributorShare
}

// ContributorShare is a single contributor's revenue allocation.
type ContributorShare struct {
	Address string
	Amount  int64
}

// CalculateRevenue computes the 4-pool revenue split from a service call payment.
// All arithmetic uses int64 with basis-point math. The last pool (protocol treasury)
// absorbs any remainder from rounding to ensure lossless distribution.
//
// If there are no contributors, the contributor pool redirects to protocol treasury.
func CalculateRevenue(
	total int64,
	evidenceTaxBp, contributorsBp, researchBp, treasuryBp uint32,
	contributors []*types.ContributorRecord,
) RevenueDistribution {
	if total <= 0 {
		return RevenueDistribution{}
	}

	dist := RevenueDistribution{}

	// Primary 4-way split
	dist.EvidenceTax = total * int64(evidenceTaxBp) / BpDenominator
	dist.ContributorPool = total * int64(contributorsBp) / BpDenominator
	dist.ResearchFund = total * int64(researchBp) / BpDenominator
	// Treasury absorbs remainder to prevent dust loss
	dist.ProtocolTreasury = total - dist.EvidenceTax - dist.ContributorPool - dist.ResearchFund

	// Evidence tax 5-way sub-split
	dist.TaxResearch = dist.EvidenceTax * TaxResearchBp / BpDenominator
	dist.TaxFoundation = dist.EvidenceTax * TaxFoundationBp / BpDenominator
	dist.TaxCommunity = dist.EvidenceTax * TaxCommunityBp / BpDenominator
	dist.TaxDevelopers = dist.EvidenceTax * TaxDevelopersBp / BpDenominator
	// Reserve absorbs evidence tax remainder
	dist.TaxReserve = dist.EvidenceTax - dist.TaxResearch - dist.TaxFoundation - dist.TaxCommunity - dist.TaxDevelopers

	// Contributor pro-rata split
	if len(contributors) == 0 {
		// No contributors: redirect pool to protocol treasury
		dist.ProtocolTreasury += dist.ContributorPool
		dist.ContributorPool = 0
		return dist
	}

	// Build sorted contributor list (deterministic ordering by address)
	sorted := make([]*types.ContributorRecord, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Did < sorted[j].Did
	})

	// Calculate total completed tasks for weighting
	var totalTasks uint32
	for _, c := range sorted {
		totalTasks += c.TasksCompleted
	}

	if totalTasks == 0 {
		// Equal split if no tasks completed yet
		perContributor := dist.ContributorPool / int64(len(sorted))
		var distributed int64
		for i, c := range sorted {
			amt := perContributor
			if i == len(sorted)-1 {
				// Last contributor absorbs remainder
				amt = dist.ContributorPool - distributed
			}
			dist.ContributorShares = append(dist.ContributorShares, ContributorShare{
				Address: c.Did,
				Amount:  amt,
			})
			distributed += amt
		}
	} else {
		// Pro-rata by tasks completed
		var distributed int64
		for i, c := range sorted {
			var amt int64
			if i == len(sorted)-1 {
				// Last contributor absorbs remainder
				amt = dist.ContributorPool - distributed
			} else {
				amt = dist.ContributorPool * int64(c.TasksCompleted) / int64(totalTasks)
			}
			dist.ContributorShares = append(dist.ContributorShares, ContributorShare{
				Address: c.Did,
				Amount:  amt,
			})
			distributed += amt
		}
	}

	return dist
}
