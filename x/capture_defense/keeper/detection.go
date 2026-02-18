package keeper

import (
	"fmt"
	"math"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// AnalyzeCaptureRisk computes capture risk metrics for a domain based on
// recent verification history. Returns updated CaptureMetrics.
func (k Keeper) AnalyzeCaptureRisk(ctx sdk.Context, domain string, currentBlock uint64) *types.CaptureMetrics {
	history := k.GetVerificationHistory(ctx, domain)

	if uint64(len(history.Entries)) < types.MinVerificationsForAnalysis {
		return &types.CaptureMetrics{
			Domain:          domain,
			AnalyzedAtBlock: currentBlock,
		}
	}

	// Count per-validator participation
	validatorCounts := make(map[string]uint64)
	var totalParticipations uint64
	for _, entry := range history.Entries {
		for _, v := range entry.Validators {
			validatorCounts[v]++
			totalParticipations++
		}
	}

	uniqueValidators := uint64(len(validatorCounts))

	// Compute Herfindahl-Hirschman Index (HHI)
	hhi := computeHHI(validatorCounts, totalParticipations)

	// Compute top-3 validator share
	top3Share := computeTop3Share(validatorCounts, totalParticipations)

	// Compute timing correlation
	timingCorr := computeTimingCorrelation(history.Entries)

	// Compute verdict correlation
	verdictCorr := computeVerdictCorrelation(history.Entries)

	// Composite risk score: weighted combination
	// HHI 40%, timing 20%, verdict 20%, top3 20%
	riskScore := (hhi*400_000 + timingCorr*200_000 + verdictCorr*200_000 + top3Share*200_000) / 1_000_000

	metrics := &types.CaptureMetrics{
		Domain:             domain,
		HerfindahlIndex:    hhi,
		TimingCorrelation:  timingCorr,
		VerdictCorrelation: verdictCorr,
		UniqueValidators:   uniqueValidators,
		TotalVerifications: uint64(len(history.Entries)),
		Top3ValidatorShare: top3Share,
		AnalyzedAtBlock:    currentBlock,
		RiskScore:          riskScore,
	}

	k.SetCaptureMetrics(ctx, metrics)
	return metrics
}

// computeHHI computes the Herfindahl-Hirschman Index (market concentration).
// HHI = sum(share_i^2) where share_i is each validator's fraction of total.
// Returns basis points (0-1,000,000). HHI of 1,000,000 = perfect monopoly.
func computeHHI(validatorCounts map[string]uint64, totalParticipations uint64) uint64 {
	if totalParticipations == 0 {
		return 0
	}

	var hhi float64
	for _, count := range validatorCounts {
		share := float64(count) / float64(totalParticipations)
		hhi += share * share
	}

	return uint64(hhi * 1_000_000)
}

// computeTop3Share computes the market share of the top 3 validators.
func computeTop3Share(validatorCounts map[string]uint64, totalParticipations uint64) uint64 {
	if totalParticipations == 0 {
		return 0
	}

	counts := make([]uint64, 0, len(validatorCounts))
	for _, c := range validatorCounts {
		counts = append(counts, c)
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i] > counts[j] })

	var top3 uint64
	for i := 0; i < 3 && i < len(counts); i++ {
		top3 += counts[i]
	}

	return (top3 * 1_000_000) / totalParticipations
}

// computeTimingCorrelation detects suspiciously correlated submission timing.
// Returns basis points (0-1,000,000). High value = suspicious coordination.
func computeTimingCorrelation(entries []*types.VerificationHistoryEntry) uint64 {
	if len(entries) < 2 {
		return 0
	}

	// For each round, compute the variance in submission timestamps.
	// Low variance across rounds suggests coordination.
	var lowVarianceCount uint64
	for _, entry := range entries {
		if len(entry.SubmissionTimestamps) < 2 {
			continue
		}

		// Compute mean
		var sum uint64
		for _, ts := range entry.SubmissionTimestamps {
			sum += ts
		}
		mean := float64(sum) / float64(len(entry.SubmissionTimestamps))

		// Compute variance
		var variance float64
		for _, ts := range entry.SubmissionTimestamps {
			diff := float64(ts) - mean
			variance += diff * diff
		}
		variance /= float64(len(entry.SubmissionTimestamps))

		// Threshold: standard deviation < 2 blocks is suspiciously coordinated
		if math.Sqrt(variance) < 2.0 {
			lowVarianceCount++
		}
	}

	if len(entries) == 0 {
		return 0
	}
	return (lowVarianceCount * 1_000_000) / uint64(len(entries))
}

// computeVerdictCorrelation detects suspiciously uniform verdicts.
// Returns basis points (0-1,000,000). High value = suspicious agreement.
func computeVerdictCorrelation(entries []*types.VerificationHistoryEntry) uint64 {
	if len(entries) == 0 {
		return 0
	}

	var unanimousCount uint64
	for _, entry := range entries {
		if len(entry.Verdicts) < 2 {
			continue
		}

		// Check if all verdicts are identical
		allSame := true
		first := entry.Verdicts[0]
		for _, v := range entry.Verdicts[1:] {
			if v != first {
				allSame = false
				break
			}
		}
		if allSame {
			unanimousCount++
		}
	}

	return (unanimousCount * 1_000_000) / uint64(len(entries))
}

// RunCaptureDetection runs detection analysis on all domains with sufficient history.
// When risk scores exceed the pause threshold, the domain is automatically paused
// for DefaultDomainPauseDurationBlocks to prevent further verification damage.
func (k Keeper) RunCaptureDetection(ctx sdk.Context, currentBlock uint64) {
	// Skip detection while chain is halted — guardians are handling the emergency
	if k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx) {
		return
	}

	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.VerificationHistoryKeyPrefix, prefixEndBytes(types.VerificationHistoryKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		domain := string(iter.Key()[len(types.VerificationHistoryKeyPrefix):])
		metrics := k.AnalyzeCaptureRisk(ctx, domain, currentBlock)

		// Emit events at threshold
		if metrics.RiskScore >= types.HerfindahlTriggerThreshold {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"capture_risk_detected",
					sdk.NewAttribute("domain", domain),
					sdk.NewAttribute("risk_score", fmt.Sprintf("%d", metrics.RiskScore)),
					sdk.NewAttribute("hhi", fmt.Sprintf("%d", metrics.HerfindahlIndex)),
				),
			)
		}

		if metrics.RiskScore >= types.HerfindahlPauseThreshold {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"capture_risk_critical",
					sdk.NewAttribute("domain", domain),
					sdk.NewAttribute("risk_score", fmt.Sprintf("%d", metrics.RiskScore)),
				),
			)

			// Auto-pause domain to prevent further verification damage
			k.PauseDomain(ctx, domain, currentBlock+types.DefaultDomainPauseDurationBlocks)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"domain_auto_paused",
					sdk.NewAttribute("domain", domain),
					sdk.NewAttribute("paused_until_block", fmt.Sprintf("%d", currentBlock+types.DefaultDomainPauseDurationBlocks)),
					sdk.NewAttribute("reason", "capture_risk_critical"),
				),
			)
		}
	}
}

// FindSuspiciousPairs identifies pairs of validators with correlated behavior.
func (k Keeper) FindSuspiciousPairs(ctx sdk.Context, domain string) [][2]string {
	history := k.GetVerificationHistory(ctx, domain)
	if len(history.Entries) < 5 {
		return nil
	}

	// Build co-occurrence and co-verdict matrix
	type pairKey struct{ a, b string }
	coOccurrence := make(map[pairKey]uint64)
	coVerdict := make(map[pairKey]uint64)

	for _, entry := range history.Entries {
		for i := 0; i < len(entry.Validators); i++ {
			for j := i + 1; j < len(entry.Validators); j++ {
				a, b := entry.Validators[i], entry.Validators[j]
				if a > b {
					a, b = b, a
				}
				pk := pairKey{a, b}
				coOccurrence[pk]++
				if i < len(entry.Verdicts) && j < len(entry.Verdicts) {
					if entry.Verdicts[i] == entry.Verdicts[j] {
						coVerdict[pk]++
					}
				}
			}
		}
	}

	var suspicious [][2]string
	for pk, coOcc := range coOccurrence {
		if coOcc < 5 {
			continue
		}
		coVerd := coVerdict[pk]
		agreementRate := (coVerd * 1_000_000) / coOcc
		// >95% agreement across 5+ co-occurrences is suspicious
		if agreementRate > 950_000 {
			suspicious = append(suspicious, [2]string{pk.a, pk.b})
		}
	}

	return suspicious
}
