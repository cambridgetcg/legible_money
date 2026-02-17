package types

import (
	"context"
)

// KnowledgeKeeper provides knowledge metrics for alignment sensors.
type KnowledgeKeeper interface {
	// Truth sensor (TDI)
	GetVerifiedFactCount(ctx context.Context) uint64
	GetSupersededFactCount(ctx context.Context) uint64
	GetTotalChallenges(ctx context.Context) uint64
	GetChallengesByTopValidators(ctx context.Context, topN int) uint64
	GetUnanimousRoundCount(ctx context.Context) uint64
	GetTotalRoundCount(ctx context.Context) uint64
	GetCounterFactCount(ctx context.Context) uint64

	// Understanding sensor (DCI)
	GetAvgCitationChainLength(ctx context.Context) uint64 // BPS
	GetCrossDomainCitationRatio(ctx context.Context) uint64 // BPS
	GetOrphanFactCount(ctx context.Context) uint64
	GetActiveFactCount(ctx context.Context) uint64
	GetAvgSupersessionChainLength(ctx context.Context) uint64 // BPS

	// Novelty sensor (NI)
	GetTotalFactCount(ctx context.Context) uint64
	GetFactsInNewCategories(ctx context.Context, epochStart, epochEnd uint64) uint64
	GetNewCrossDomainEdges(ctx context.Context, epochStart, epochEnd uint64) uint64
	GetDomainFactDistribution(ctx context.Context) []DomainFactCount

	// Fairness sensor (FI)
	GetNewSubmitterAcceptRate(ctx context.Context) uint64 // BPS
}

// StakingKeeper provides staking metrics for alignment sensors.
type StakingKeeper interface {
	// Truth sensor (TDI)
	GetAvgPairwiseAgreement(ctx context.Context) uint64 // BPS

	// Fairness sensor (FI)
	GetValidatorRewardDistribution(ctx context.Context) []ValidatorReward
	GetSlashProportionality(ctx context.Context) uint64 // BPS
	GetTop10StakeRatio(ctx context.Context) uint64       // BPS
	GetTotalStake(ctx context.Context) uint64
}

// OntologyKeeper provides ontology metrics for alignment sensors.
type OntologyKeeper interface {
	// Understanding sensor (DCI)
	GetActiveDomainCount(ctx context.Context) uint64
	GetDomainsWithActivity(ctx context.Context, epochStart, epochEnd uint64) uint64

	// Novelty sensor (NI)
	GetNewDomainsInEpoch(ctx context.Context, epochStart, epochEnd uint64) uint64
	GetTotalDomainCount(ctx context.Context) uint64
}

// AutopoiesisKeeper reads/writes multipliers and adaptation state.
type AutopoiesisKeeper interface {
	// Multiplier access
	GetMultiplier(ctx context.Context, path string) uint64
	SetMultiplier(ctx context.Context, entry MultiplierEntry)
	GetMultiplierEntry(ctx context.Context, path string) *MultiplierEntry

	// Harmony sensor (HI) — reads autopoiesis internal state
	GetAllMultiplierEntries(ctx context.Context) []MultiplierEntry
	GetAdaptationCount(ctx context.Context, epoch uint64) uint64
	GetConflictCount(ctx context.Context, epoch uint64) uint64
	GetDirectionChanges(ctx context.Context, lastNEpochs uint64) uint64

	// Emergency check
	IsActive(ctx context.Context) bool
}

// EmergencyKeeper checks whether the chain is in emergency halt.
type EmergencyKeeper interface {
	IsHalted(ctx context.Context) bool
}

// VestingRewardsKeeper provides reward distribution data for fairness sensor.
type VestingRewardsKeeper interface {
	GetValidatorRewardsInEpoch(ctx context.Context, epochStart, epochEnd uint64) []ValidatorReward
}

// DomainFactCount holds fact count per domain for diversity calculation.
type DomainFactCount struct {
	Domain string `json:"domain"`
	Count  uint64 `json:"count"`
}

// ValidatorReward holds per-validator reward for Gini calculation.
type ValidatorReward struct {
	Validator string `json:"validator"`
	Amount    uint64 `json:"amount"` // ulgm
}

// MultiplierEntry mirrors autopoiesis MultiplierEntry for alignment writes.
// Imported to avoid circular dependency — alignment writes these values
// into the autopoiesis store via the AutopoiesisKeeper interface.
type MultiplierEntry struct {
	Path    string `json:"path"`
	Value   uint64 `json:"value"`   // BPS (1M = 1.0x)
	Hormone string `json:"hormone"` // "ALIGNMENT" for alignment-driven entries
	Epoch   uint64 `json:"epoch"`
}
