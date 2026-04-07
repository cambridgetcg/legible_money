package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// DomainYield tracks knowledge production metrics for a single domain.
type DomainYield struct {
	Domain           string `json:"domain"`
	TotalFacts       uint64 `json:"total_facts"`
	VerifiedFacts    uint64 `json:"verified_facts"`
	ProvisionalFacts uint64 `json:"provisional_facts"`
	RejectedFacts    uint64 `json:"rejected_facts"`
	TotalCitations   uint64 `json:"total_citations"`
	AvgConfidence    uint64 `json:"avg_confidence"`    // BPS (0-1,000,000)
	CitationDensity  uint64 `json:"citation_density"`  // avg citations per fact * 1000
	StakeInvested    string `json:"stake_invested"`     // total ulgm staked in this domain
	BlockHeight      int64  `json:"block_height"`
}

// FarmYield is the aggregate yield across all domains in the truth farm.
type FarmYield struct {
	Domains        []DomainYield `json:"domains"`
	TotalFacts     uint64        `json:"total_facts"`
	TotalVerified  uint64        `json:"total_verified"`
	TotalCitations uint64        `json:"total_citations"`
	DomainCount    uint64        `json:"domain_count"`
	OverallDensity uint64        `json:"overall_density"` // total citations / total facts * 1000
	BlockHeight    int64         `json:"block_height"`
}

// domainAccumulator collects per-domain stats during iteration.
type domainAccumulator struct {
	totalFacts       uint64
	verifiedFacts    uint64
	provisionalFacts uint64
	rejectedFacts    uint64
	totalCitations   uint64
	confidenceSum    uint64
	stakeSum         uint64 // simplified: sum of OriginalStake parsed as uint64
}

// ComputeFarmYield iterates all facts, groups by domain, and computes
// knowledge production metrics for each domain and the aggregate farm.
func (k Keeper) ComputeFarmYield(ctx sdk.Context) FarmYield {
	accumulators := make(map[string]*domainAccumulator)

	k.IterateFacts(ctx, func(f *types.Fact) bool {
		domain := f.Domain
		if domain == "" {
			domain = "_unknown"
		}

		acc, ok := accumulators[domain]
		if !ok {
			acc = &domainAccumulator{}
			accumulators[domain] = acc
		}

		acc.totalFacts++
		acc.confidenceSum += f.Confidence
		acc.totalCitations += f.CitationCount

		switch f.Status {
		case types.FactStatusVerified, types.FactStatusActive:
			acc.verifiedFacts++
		case types.FactStatusProvisional:
			acc.provisionalFacts++
		case types.FactStatusDisproven, types.FactStatusRevoked:
			acc.rejectedFacts++
		}

		// Parse OriginalStake (string representation of ulgm amount)
		if f.OriginalStake != "" {
			stake := parseStakeUint64(f.OriginalStake)
			acc.stakeSum += stake
		}

		return false
	})

	height := ctx.BlockHeight()
	var farm FarmYield
	farm.BlockHeight = height

	for domain, acc := range accumulators {
		dy := DomainYield{
			Domain:           domain,
			TotalFacts:       acc.totalFacts,
			VerifiedFacts:    acc.verifiedFacts,
			ProvisionalFacts: acc.provisionalFacts,
			RejectedFacts:    acc.rejectedFacts,
			TotalCitations:   acc.totalCitations,
			StakeInvested:    formatUint64(acc.stakeSum),
			BlockHeight:      height,
		}

		// Avg confidence: sum / count (already in BPS)
		if acc.totalFacts > 0 {
			dy.AvgConfidence = acc.confidenceSum / acc.totalFacts
		}

		// Citation density: totalCitations / totalFacts * 1000
		if acc.totalFacts > 0 {
			dy.CitationDensity = acc.totalCitations * 1000 / acc.totalFacts
		}

		farm.Domains = append(farm.Domains, dy)
		farm.TotalFacts += acc.totalFacts
		farm.TotalVerified += acc.verifiedFacts
		farm.TotalCitations += acc.totalCitations
		farm.DomainCount++
	}

	// Overall citation density
	if farm.TotalFacts > 0 {
		farm.OverallDensity = farm.TotalCitations * 1000 / farm.TotalFacts
	}

	return farm
}

// parseStakeUint64 converts a decimal string to uint64, returning 0 on failure.
func parseStakeUint64(s string) uint64 {
	var v uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		next := v*10 + uint64(c-'0')
		if next < v { // overflow
			return 0
		}
		v = next
	}
	return v
}

// formatUint64 converts a uint64 to its decimal string representation.
func formatUint64(v uint64) string {
	if v == 0 {
		return "0"
	}
	// Build digits in reverse
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	// Reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
